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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace     = "crossplane-system"
	readyTimeout  = 3 * time.Minute
	deleteTimeout = 2 * time.Minute
	settleWindow  = 15 * time.Second
)

var (
	secretGVK = schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
	caGVK     = schema.GroupVersionKind{Group: "integrations.groundcover.com", Version: "v1alpha1", Kind: "ConnectedAppJson"}
	nrGVK     = schema.GroupVersionKind{Group: "notifications.groundcover.com", Version: "v1alpha1", Kind: "NotificationRoute"}
)

// TestNotificationRouteLifecycle creates a ConnectedAppJson, references it from a
// NotificationRoute, asserts both reach Ready, and confirms the route stays Synced (no
// drift). It exercises the notification_route nested-attribute path end to end.
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

	// No drift: after settling, the route must still report Synced=True with nothing to push.
	time.Sleep(settleWindow)
	if !conditionTrue(ctx, t, cl, nrGVK, name, "Synced") {
		t.Fatal("NotificationRoute is not Synced after settling — possible drift")
	}
	t.Log("E2E OK: ConnectedAppJson + NotificationRoute Ready, route Synced (no drift)")
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
