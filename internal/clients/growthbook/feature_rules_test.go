/*
Copyright 2026 The provider-growthbook Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package growthbook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeatureRuleEncodingForce(t *testing.T) {
	rules := []FeatureRule{{
		Type:            "force",
		Description:     "force to US",
		Enabled:         featurePtrBool(true),
		Condition:       `{"country":"US"}`,
		AllEnvironments: true,
		Value:           "true",
	}}

	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(featureEnvelope{Feature: Feature{ID: "ft_1", ValueType: "boolean", DefaultValue: "true"}})
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	if _, err := c.UpdateFeature(context.Background(), "ft_1", FeatureRequest{Rules: &rules}); err != nil {
		t.Fatalf("UpdateFeature() error = %v", err)
	}

	rawRules, _ := gotBody["rules"].([]any)
	if len(rawRules) != 1 {
		t.Fatalf("rules = %+v, want 1 entry", gotBody["rules"])
	}
	rule, _ := rawRules[0].(map[string]any)
	want := map[string]any{
		"type":            "force",
		"description":     "force to US",
		"enabled":         true,
		"condition":       `{"country":"US"}`,
		"allEnvironments": true,
		"value":           "true",
	}
	for k, v := range want {
		if rule[k] != v {
			t.Errorf("force rule[%q] = %#v, want %#v", k, rule[k], v)
		}
	}
	for _, forbidden := range []string{"coverage", "hashAttribute", "experimentId", "variations", "environments", "id"} {
		if _, ok := rule[forbidden]; ok {
			t.Errorf("force rule must not encode %q, got %#v", forbidden, rule[forbidden])
		}
	}
}

func TestFeatureRuleEncodingRollout(t *testing.T) {
	coverage := 0.25
	rules := []FeatureRule{{
		Type:            "rollout",
		AllEnvironments: false,
		Environments:    []string{"staging"},
		Value:           "on",
		Coverage:        &coverage,
		HashAttribute:   "id",
	}}

	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(featureEnvelope{Feature: Feature{ID: "ft_1", ValueType: "string", DefaultValue: "off"}})
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	if _, err := c.UpdateFeature(context.Background(), "ft_1", FeatureRequest{Rules: &rules}); err != nil {
		t.Fatalf("UpdateFeature() error = %v", err)
	}

	rawRules, _ := gotBody["rules"].([]any)
	rule, _ := rawRules[0].(map[string]any)
	if rule["type"] != "rollout" {
		t.Errorf("type = %#v, want rollout", rule["type"])
	}
	if rule["allEnvironments"] != false {
		t.Errorf("allEnvironments = %#v, want false", rule["allEnvironments"])
	}
	envs, _ := rule["environments"].([]any)
	if len(envs) != 1 || envs[0] != "staging" {
		t.Errorf("environments = %#v, want [staging]", rule["environments"])
	}
	// coverage must round-trip as a JSON number, not a string.
	cov, ok := rule["coverage"].(float64)
	if !ok || cov != 0.25 {
		t.Errorf("coverage = %#v (%T), want number 0.25", rule["coverage"], rule["coverage"])
	}
	if rule["hashAttribute"] != "id" {
		t.Errorf("hashAttribute = %#v, want id", rule["hashAttribute"])
	}
}

func TestFeatureRuleEncodingExperimentRef(t *testing.T) {
	rules := []FeatureRule{{
		Type:            "experiment-ref",
		AllEnvironments: true,
		ExperimentID:    "exp_1",
		Variations: []FeatureRuleVariation{
			{VariationID: "0", Value: "control"},
			{VariationID: "1", Value: "treatment"},
		},
	}}

	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(featureEnvelope{Feature: Feature{ID: "ft_1", ValueType: "string", DefaultValue: "control"}})
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	if _, err := c.UpdateFeature(context.Background(), "ft_1", FeatureRequest{Rules: &rules}); err != nil {
		t.Fatalf("UpdateFeature() error = %v", err)
	}

	rawRules, _ := gotBody["rules"].([]any)
	rule, _ := rawRules[0].(map[string]any)
	if rule["type"] != "experiment-ref" || rule["experimentId"] != "exp_1" {
		t.Fatalf("rule = %+v", rule)
	}
	variations, _ := rule["variations"].([]any)
	if len(variations) != 2 {
		t.Fatalf("variations = %+v, want 2", rule["variations"])
	}
	v0, _ := variations[0].(map[string]any)
	if v0["variationId"] != "0" || v0["value"] != "control" {
		t.Errorf("variations[0] = %+v", v0)
	}
	if _, ok := rule["value"]; ok {
		t.Errorf("experiment-ref rule must not encode top-level value, got %#v", rule["value"])
	}
}

func TestFeatureRequestRulesNilVsEmpty(t *testing.T) {
	// A nil Rules pointer omits the field entirely.
	b, err := json.Marshal(FeatureRequest{Description: ptrStr("no rule change")})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := m["rules"]; ok {
		t.Errorf("nil Rules must omit the field, got %s", b)
	}

	// A non-nil, empty Rules slice clears the array by encoding [].
	empty := []FeatureRule{}
	b, err = json.Marshal(FeatureRequest{Rules: &empty})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	rawRules, ok := m["rules"].([]any)
	if !ok || len(rawRules) != 0 {
		t.Errorf("empty non-nil Rules must encode as [], got %s", b)
	}
}

func TestFeatureGetDecodesRules(t *testing.T) {
	coverage := 0.5
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(featureEnvelope{Feature: Feature{
			ID:           "ft_1",
			ValueType:    "boolean",
			DefaultValue: "true",
			Rules: []FeatureRule{
				{Type: "force", ID: "fr_1", AllEnvironments: true, Value: "true"},
				{Type: "rollout", ID: "fr_2", AllEnvironments: false, Environments: []string{"production"}, Value: "true", Coverage: &coverage, HashAttribute: "id"},
			},
		}})
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	f, err := c.GetFeature(context.Background(), "ft_1")
	if err != nil {
		t.Fatalf("GetFeature() error = %v", err)
	}
	if len(f.Rules) != 2 {
		t.Fatalf("Rules = %+v, want 2 entries", f.Rules)
	}
	if f.Rules[0].Type != "force" || f.Rules[0].ID != "fr_1" {
		t.Errorf("Rules[0] = %+v", f.Rules[0])
	}
	if f.Rules[1].Type != "rollout" || f.Rules[1].Coverage == nil || *f.Rules[1].Coverage != 0.5 {
		t.Errorf("Rules[1] = %+v", f.Rules[1])
	}
}
