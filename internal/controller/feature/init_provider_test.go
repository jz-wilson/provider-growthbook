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

package feature

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

func withInitDescription(d string) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.InitProvider.Description = ptr(d) }
}

func withInitRules(rules ...v1alpha1.FeatureRule) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.InitProvider.Rules = rules }
}

func initForceRule(value string) v1alpha1.FeatureRule {
	return v1alpha1.FeatureRule{Type: ruleTypeForce, Value: ptr(value)}
}

// TestCreateInitProvider covers the spec.initProvider merge semantics on
// Create: a field set only in initProvider is sent (including rules), and
// forProvider wins when both are set.
func TestCreateInitProvider(t *testing.T) {
	cases := map[string]struct {
		reason      string
		cr          *v1alpha1.Feature
		wantDesc    *string
		wantRuleLen int
	}{
		"InitProviderOnlyFieldIsSent": {
			reason:      "A field set only in initProvider must be sent on Create.",
			cr:          newFeature(withInitDescription("init desc"), withInitRules(initForceRule("true"))),
			wantDesc:    ptr("init desc"),
			wantRuleLen: 1,
		},
		"ForProviderOverridesInitProvider": {
			reason:      "forProvider wins over initProvider when both set the same field.",
			cr:          newFeature(withInitDescription("init desc"), withDescription("forProvider desc")),
			wantDesc:    ptr("forProvider desc"),
			wantRuleLen: 0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var gotReq growthbook.FeatureRequest
			client := &fakeClient{create: func(_ context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
				gotReq = req
				return &growthbook.Feature{ID: req.ID, ValueType: req.ValueType, DefaultValue: req.DefaultValue}, nil
			}}
			e := external{client: client}
			if _, err := e.Create(context.Background(), tc.cr); err != nil {
				t.Fatalf("e.Create(...): unexpected error %v", err)
			}
			if diff := cmp.Diff(tc.wantDesc, gotReq.Description); diff != "" {
				t.Errorf("\n%s\ndescription: -want, +got:\n%s", tc.reason, diff)
			}
			gotRuleLen := 0
			if gotReq.Rules != nil {
				gotRuleLen = len(*gotReq.Rules)
			}
			if gotRuleLen != tc.wantRuleLen {
				t.Errorf("\n%s\nrules len = %d, want %d", tc.reason, gotRuleLen, tc.wantRuleLen)
			}
		})
	}
}

// TestObserveIgnoresInitProviderDrift asserts that a later API-side change
// to a field or rule set that was only ever set via initProvider never
// marks the resource out of date, and is not copied into initProvider by
// late-initialization.
func TestObserveIgnoresInitProviderDrift(t *testing.T) {
	cr := newFeature(withInitDescription("init desc"), withInitRules(initForceRule("true")))
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) {
		return &growthbook.Feature{
			ID: featureID, ValueType: valTypeBoolean, DefaultValue: valTrue,
			Description: "changed out from under us",
			Rules:       []growthbook.FeatureRule{{Type: ruleTypeForce, Value: valFalse}, {Type: ruleTypeForce, Value: "also different"}},
		}, nil
	}}

	e := external{client: client}
	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("e.Observe(...): unexpected error %v", err)
	}
	if !got.ResourceUpToDate {
		t.Errorf("initProvider-only fields/rules must never cause drift, got %+v", got)
	}
	if cr.Spec.InitProvider.Description == nil || *cr.Spec.InitProvider.Description != "init desc" {
		t.Errorf("late-initialization must not overwrite initProvider description: got %v", cr.Spec.InitProvider.Description)
	}
	if len(cr.Spec.InitProvider.Rules) != 1 {
		t.Errorf("late-initialization must not overwrite initProvider rules: got %v", cr.Spec.InitProvider.Rules)
	}
}

// TestUpdateNeverSendsInitProviderOnlyValues asserts Update never leaks
// initProvider-only field or rule values into the request body.
func TestUpdateNeverSendsInitProviderOnlyValues(t *testing.T) {
	var gotReq growthbook.FeatureRequest
	client := &fakeClient{update: func(_ context.Context, _ string, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
		gotReq = req
		return &growthbook.Feature{ID: featureID, ValueType: valTypeBoolean, DefaultValue: valTrue}, nil
	}}
	cr := newFeature(withInitDescription("init desc"), withInitRules(initForceRule("true")))

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotReq.Description != nil {
		t.Errorf("Update must not send initProvider-only description: %+v", gotReq)
	}
	if gotReq.Rules != nil {
		t.Errorf("Update must not send initProvider-only rules: %+v", gotReq)
	}
}
