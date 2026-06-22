package config

import (
	"encoding/json"
	_ "embed"
)

// schemaJSON is the Terraform provider schema, produced by `make schema`
// (`terraform providers schema -json`). It is embedded so both the generator and the
// runtime provider build the same upjet config.Provider from the same schema.
//
//go:embed schema.json
var schemaJSON []byte

// coerceDynamicAttributesToString rewrites every attribute whose Terraform type is the
// dynamic pseudo-type to "string" across all resource, data-source, and provider blocks.
//
// upjet's NewProvider converts the whole schema to an SDKv2 resource map and panics on
// cty DynamicPseudoType attributes (e.g. connected_app.data, a JSON object). Coercing
// dynamic attributes to JSON strings — at both generation and runtime — makes the CRD
// carry the data as an opaque JSON string, which is how a Crossplane user supplies it
// and pairs with the hash-based observe strategy for that resource.
func coerceDynamicAttributesToString(schema []byte) []byte {
	var doc map[string]any
	if err := json.Unmarshal(schema, &doc); err != nil {
		panic("parse provider schema JSON: " + err.Error())
	}

	providerSchemas, _ := doc["provider_schemas"].(map[string]any)
	for _, ps := range providerSchemas {
		psMap, ok := ps.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"resource_schemas", "data_source_schemas"} {
			schemas, ok := psMap[key].(map[string]any)
			if !ok {
				continue
			}
			for _, rs := range schemas {
				if rsMap, ok := rs.(map[string]any); ok {
					coerceBlock(rsMap["block"])
				}
			}
		}
		if prov, ok := psMap["provider"].(map[string]any); ok {
			coerceBlock(prov["block"])
		}
	}

	out, err := json.Marshal(doc)
	if err != nil {
		panic("re-marshal provider schema JSON: " + err.Error())
	}
	return out
}

func coerceBlock(block any) {
	b, ok := block.(map[string]any)
	if !ok {
		return
	}
	if attrs, ok := b["attributes"].(map[string]any); ok {
		for _, attr := range attrs {
			am, ok := attr.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := am["type"].(string); ok && t == "dynamic" {
				// upjet cannot construct a DynamicPseudoType value at runtime from a flat
				// param, and a *sensitive* dynamic becomes a secret-ref string — which
				// tftypes.ValueFromJSON can never decode into a dynamic. Represent it as an
				// inline map(string) instead: upjet emits a real JSON object, which
				// ValueFromJSON infers as an object and the provider's dynamic attribute
				// accepts. ponytail: map(string) only covers flat data (e.g. slack-webhook
				// {url}); nested configs need a richer shape — revisit when those matter.
				am["type"] = []any{"map", "string"}
				// Drop sensitivity so upjet keeps the value inline rather than turning it
				// into a secret-ref string (which would reintroduce the dynamic mismatch).
				delete(am, "sensitive")
			}
		}
	}
	if blockTypes, ok := b["block_types"].(map[string]any); ok {
		for _, bt := range blockTypes {
			if btMap, ok := bt.(map[string]any); ok {
				coerceBlock(btMap["block"])
			}
		}
	}
}
