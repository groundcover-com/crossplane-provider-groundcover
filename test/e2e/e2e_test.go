//go:build e2e

// Package e2e is a live-backend end-to-end test for the groundcover Crossplane provider.
//
// It runs against whatever cluster the current kubeconfig points at (the CI workflow spins
// a kind cluster, installs the CRDs, configures a ProviderConfig + credentials Secret, and
// runs the provider out-of-cluster). The test drives the resources through the Kubernetes
// API as a user would — create, wait for Ready/Synced, confirm no drift, delete — and uses
// unstructured objects so it does not depend on the generated API types.
//
// Build-tagged `e2e` so it is excluded from the normal `go test ./...`. Run with:
//
//	go test -tags e2e -v -timeout 15m ./test/e2e/...
package e2e

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace     = "crossplane-system"
	readyTimeout  = 3 * time.Minute
	deleteTimeout = 2 * time.Minute
	settleWindow  = 45 * time.Second // must span several --poll cycles (provider runs --poll=10s)
)

var (
	secretGVK = schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
	caGVK     = schema.GroupVersionKind{Group: "integrations.groundcover.com", Version: "v1alpha1", Kind: "ConnectedAppJson"}
	nrGVK     = schema.GroupVersionKind{Group: "notifications.groundcover.com", Version: "v1alpha1", Kind: "NotificationRoute"}
	policyGVK = schema.GroupVersionKind{Group: "rbac.groundcover.com", Version: "v1alpha1", Kind: "Policy"}
)

// TestNotificationRouteLifecycle creates a ConnectedAppJson, references it from a
// NotificationRoute, asserts both reach Ready, confirms no drift, then updates a field and
// confirms the change is applied and re-converges (no drift after update), and finally
// deletes. It exercises the notification_route nested-attribute path end to end.
func TestNotificationRouteLifecycle(t *testing.T) {
	ctx := context.Background()
	cl := newClient(t)

	id := runID()
	name := "e2e-ci-" + id

	// Data secret backing the connected app (namespaced).
	secret := newObject(secretGVK, name+"-data")
	secret.SetNamespace(namespace)
	secret.Object["stringData"] = map[string]any{
		"data": `{"url":"https://hooks.slack.com/services/T0/B0/XXXXXXXXXXXX"}`,
	}
	create(ctx, t, cl, secret)
	t.Cleanup(func() { _ = cl.Delete(context.Background(), secret) })

	// ConnectedAppJson (cluster-scoped managed resource).
	ca := newObject(caGVK, name)
	ca.Object["spec"] = map[string]any{
		"providerConfigRef": map[string]any{"name": "default"},
		"forProvider": map[string]any{
			"name": name,
			"type": "slack-webhook",
			"dataSecretRef": map[string]any{
				"namespace": namespace,
				"name":      name + "-data",
				"key":       "data",
			},
		},
	}
	create(ctx, t, cl, ca)
	t.Cleanup(func() { deleteAndWait(t, cl, caGVK, name) })

	waitCondition(ctx, t, cl, caGVK, name, "Ready")
	caID := externalName(ctx, t, cl, caGVK, name)
	if caID == "" {
		t.Fatal("ConnectedAppJson has no crossplane.io/external-name annotation")
	}
	t.Logf("connected app external-name: %s", caID)

	// NotificationRoute referencing the connected app.
	nr := newObject(nrGVK, name)
	nr.Object["spec"] = map[string]any{
		"providerConfigRef": map[string]any{"name": "default"},
		"forProvider": map[string]any{
			"name":  name,
			"query": "severity = 'S1'",
			"routes": []any{
				map[string]any{
					"status":        []any{"Alerting", "Resolved"},
					"connectedApps": []any{map[string]any{"type": "slack-webhook", "id": caID}},
				},
			},
			"notificationSettings": map[string]any{"renotificationInterval": "1h"},
		},
	}
	create(ctx, t, cl, nr)
	t.Cleanup(func() { deleteAndWait(t, cl, nrGVK, name) })

	waitCondition(ctx, t, cl, nrGVK, name, "Ready")

	// No drift: a converged resource is Observed up-to-date every reconcile and never
	// re-applied. A perpetual diff (the classic upjet failure — the API returns a field in a
	// different shape than written) re-applies on every loop, emitting an
	// UpdatedExternalResource event each time. Synced=True does NOT catch this: each apply
	// succeeds, so Synced stays True while the provider thrashes. So we count update events
	// across the settle window instead.
	nr2, err := get(ctx, cl, nrGVK, name)
	if err != nil {
		t.Fatalf("get %s/%s: %v", nrGVK.Kind, name, err)
	}
	before := countUpdateEvents(ctx, t, cl, nr2)
	time.Sleep(settleWindow)
	after := countUpdateEvents(ctx, t, cl, nr2)

	if !conditionTrue(ctx, t, cl, nrGVK, name, "Synced") {
		t.Fatal("NotificationRoute is not Synced after settling")
	}
	if after > before {
		t.Fatalf("NotificationRoute drifted: %d UpdatedExternalResource event(s) during %s settle window (perpetual diff)", after-before, settleWindow)
	}
	t.Logf("create OK: Ready, Synced, no re-apply over %s (0 update events) — no drift", settleWindow)

	// Update path: change a field, confirm the provider applies it exactly as an update
	// (not a no-op), then confirm it stops re-applying — i.e. the new value round-trips
	// without producing a perpetual diff. This is the most common place upjet providers
	// break, and create-only coverage never touches it.
	baseline := countUpdateEvents(ctx, t, cl, nr2)
	setField(ctx, t, cl, nrGVK, name, "2h", "spec", "forProvider", "notificationSettings", "renotificationInterval")
	waitApplied(ctx, t, cl, nr2, baseline)
	waitUpdateConverged(ctx, t, cl, nr2, nrGVK, name, settleWindow)
	t.Logf("update OK: field applied and re-converged, no drift after update")

	t.Log("E2E OK: ConnectedAppJson + NotificationRoute create, update, no drift")
}

// TestPolicyNoDataScopeLifecycle creates a Policy that omits dataScope — the exact case a
// customer hit (BE-2586) — and asserts it reaches Ready/Synced and does not drift. It guards
// two things that only show up against a live backend:
//   - the provider (TF v1.21.0+) accepts an omitted dataScope and creates an allow-all policy,
//     instead of erroring on create;
//   - no perpetual diff: an omitted dataScope must round-trip cleanly even though the backend
//     may echo back a populated allow-all scope. That mismatch is the classic upjet failure and
//     Synced=True alone doesn't catch it, so we count UpdatedExternalResource events across the
//     settle window (same technique as the NotificationRoute test).
//
// The CRD-admission half of the fix (dropping the "dataScope is a required parameter" CEL rule)
// is what lets this manifest be created at all; this test is the runtime half.
func TestPolicyNoDataScopeLifecycle(t *testing.T) {
	ctx := context.Background()
	cl := newClient(t)

	name := "e2e-ci-policy-" + runID()

	// Policy WITHOUT dataScope (cluster-scoped managed resource). role map key must be one of
	// read/write/admin; the value is unused by the backend.
	pol := newObject(policyGVK, name)
	pol.Object["spec"] = map[string]any{
		"providerConfigRef": map[string]any{"name": "default"},
		"forProvider": map[string]any{
			"name":        name,
			"description": "e2e policy without dataScope (allow all)",
			"claimRole":   name + "-claim",
			"role":        map[string]any{"admin": "admin"},
		},
	}
	create(ctx, t, cl, pol)
	t.Cleanup(func() { deleteAndWait(t, cl, policyGVK, name) })

	// Reaching Ready proves the backend accepted the create with no dataScope (allow-all).
	waitCondition(ctx, t, cl, policyGVK, name, "Ready")

	// No drift: an omitted dataScope must not produce a perpetual diff against whatever scope
	// the backend returns. Count update events across the settle window (see NotificationRoute
	// test for why Synced=True is insufficient).
	pol2, err := get(ctx, cl, policyGVK, name)
	if err != nil {
		t.Fatalf("get %s/%s: %v", policyGVK.Kind, name, err)
	}
	before := countUpdateEvents(ctx, t, cl, pol2)
	time.Sleep(settleWindow)
	after := countUpdateEvents(ctx, t, cl, pol2)

	if !conditionTrue(ctx, t, cl, policyGVK, name, "Synced") {
		t.Fatal("Policy is not Synced after settling")
	}
	if after > before {
		t.Fatalf("Policy drifted: %d UpdatedExternalResource event(s) during %s settle window (perpetual diff on omitted dataScope)", after-before, settleWindow)
	}

	t.Log("E2E OK: Policy without dataScope create, Ready, Synced, no drift (allow-all)")
}

func runID() string {
	if v := os.Getenv("E2E_RUN_ID"); v != "" {
		return v
	}
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func newClient(t *testing.T) client.Client {
	t.Helper()
	cfg, err := ctrl.GetConfig()
	if err != nil {
		t.Fatalf("load kubeconfig: %v", err)
	}
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	return cl
}

func newObject(gvk schema.GroupVersionKind, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	u.SetName(name)
	return u
}

func create(ctx context.Context, t *testing.T, cl client.Client, u *unstructured.Unstructured) {
	t.Helper()
	if err := cl.Create(ctx, u); err != nil {
		t.Fatalf("create %s/%s: %v", u.GetKind(), u.GetName(), err)
	}
}

func get(ctx context.Context, cl client.Client, gvk schema.GroupVersionKind, name string) (*unstructured.Unstructured, error) {
	u := newObject(gvk, name)
	err := cl.Get(ctx, client.ObjectKey{Name: name}, u)
	return u, err
}

func waitCondition(ctx context.Context, t *testing.T, cl client.Client, gvk schema.GroupVersionKind, name, condType string) {
	t.Helper()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, readyTimeout, true, func(ctx context.Context) (bool, error) {
		u, err := get(ctx, cl, gvk, name)
		if err != nil {
			return false, nil
		}
		return hasConditionTrue(u, condType), nil
	})
	if err != nil {
		u, _ := get(ctx, cl, gvk, name)
		t.Fatalf("%s/%s did not reach %s=True within %s; conditions=%v", gvk.Kind, name, condType, readyTimeout, conditionsOf(u))
	}
}

func conditionTrue(ctx context.Context, t *testing.T, cl client.Client, gvk schema.GroupVersionKind, name, condType string) bool {
	t.Helper()
	u, err := get(ctx, cl, gvk, name)
	if err != nil {
		t.Fatalf("get %s/%s: %v", gvk.Kind, name, err)
	}
	return hasConditionTrue(u, condType)
}

func hasConditionTrue(u *unstructured.Unstructured, condType string) bool {
	for _, c := range conditionsOf(u) {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] == condType && m["status"] == "True" {
			return true
		}
	}
	return false
}

func conditionsOf(u *unstructured.Unstructured) []any {
	if u == nil {
		return nil
	}
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	return conds
}

// countUpdateEvents totals Crossplane "UpdatedExternalResource" events for the given managed
// resource. Kubernetes aggregates repeated identical events into one object with a rising
// count field, so we sum count (not len). Events are listed across all namespaces and matched
// by involvedObject UID — the MR is cluster-scoped, so its events land in "default".
func countUpdateEvents(ctx context.Context, t *testing.T, cl client.Client, u *unstructured.Unstructured) int {
	t.Helper()
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "EventList"})
	if err := cl.List(ctx, list); err != nil {
		t.Fatalf("list events: %v", err)
	}
	uid := string(u.GetUID())
	total := 0
	for i := range list.Items {
		e := list.Items[i].Object
		reason, _, _ := unstructured.NestedString(e, "reason")
		ioUID, _, _ := unstructured.NestedString(e, "involvedObject", "uid")
		if reason != "UpdatedExternalResource" || ioUID != uid {
			continue
		}
		if c, found, _ := unstructured.NestedInt64(e, "count"); found {
			total += int(c)
		} else {
			total++
		}
	}
	return total
}

// setField updates a single spec field on the managed resource, retrying on the optimistic
// conflict that the provider's own status writes commonly cause.
func setField(ctx context.Context, t *testing.T, cl client.Client, gvk schema.GroupVersionKind, name, value string, fields ...string) {
	t.Helper()
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		u, err := get(ctx, cl, gvk, name)
		if err != nil {
			return err
		}
		if err := unstructured.SetNestedField(u.Object, value, fields...); err != nil {
			return err
		}
		return cl.Update(ctx, u)
	})
	if err != nil {
		t.Fatalf("update %s/%s %v: %v", gvk.Kind, name, fields, err)
	}
}

// waitApplied blocks until the provider has applied at least one update beyond baseline —
// proving the spec change actually triggered an Update (not a silently-dropped no-op).
func waitApplied(ctx context.Context, t *testing.T, cl client.Client, u *unstructured.Unstructured, baseline int) {
	t.Helper()
	err := wait.PollUntilContextTimeout(ctx, 3*time.Second, readyTimeout, true, func(ctx context.Context) (bool, error) {
		return countUpdateEvents(ctx, t, cl, u) > baseline, nil
	})
	if err != nil {
		t.Fatal("provider never applied the update (no new UpdatedExternalResource event) — update path not exercised or change was dropped")
	}
}

// waitUpdateConverged blocks until the resource is Synced AND has gone quiet — no new
// UpdatedExternalResource events for `quiet` (longer than the poll interval). A clean update
// applies a bounded number of times then stops; a perpetual diff never goes quiet, so this
// times out and fails.
func waitUpdateConverged(ctx context.Context, t *testing.T, cl client.Client, u *unstructured.Unstructured, gvk schema.GroupVersionKind, name string, quiet time.Duration) {
	t.Helper()
	last := countUpdateEvents(ctx, t, cl, u)
	stableSince := time.Now()
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Second)
		n := countUpdateEvents(ctx, t, cl, u)
		if n != last {
			last = n
			stableSince = time.Now()
			continue
		}
		if conditionTrue(ctx, t, cl, gvk, name, "Synced") && time.Since(stableSince) >= quiet {
			return
		}
	}
	t.Fatalf("%s/%s never stopped re-applying after update within %s (perpetual diff)", gvk.Kind, name, readyTimeout)
}

func externalName(ctx context.Context, t *testing.T, cl client.Client, gvk schema.GroupVersionKind, name string) string {
	t.Helper()
	u, err := get(ctx, cl, gvk, name)
	if err != nil {
		t.Fatalf("get %s/%s: %v", gvk.Kind, name, err)
	}
	return u.GetAnnotations()["crossplane.io/external-name"]
}

func deleteAndWait(t *testing.T, cl client.Client, gvk schema.GroupVersionKind, name string) {
	t.Helper()
	ctx := context.Background()
	if err := cl.Delete(ctx, newObject(gvk, name)); err != nil && !apierrors.IsNotFound(err) {
		t.Errorf("delete %s/%s: %v", gvk.Kind, name, err)
		return
	}
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, deleteTimeout, true, func(ctx context.Context) (bool, error) {
		_, err := get(ctx, cl, gvk, name)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("%s/%s not deleted from backend within %s", gvk.Kind, name, deleteTimeout)
	}
}
