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

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

func withRules(rules ...v1alpha1.FeatureRule) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.ForProvider.Rules = rules }
}

func withEmptyRules() func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.ForProvider.Rules = []v1alpha1.FeatureRule{} }
}

// TestObserveRules exercises rule drift detection and status population
// through the full Observe path.
func TestObserveRules(t *testing.T) {
	cases := map[string]struct {
		reason string
		remote []growthbook.FeatureRule
		cr     *v1alpha1.Feature
		want   managed.ExternalObservation
	}{
		"NilRulesNeverDrift": {
			reason: "Rules left nil are unmanaged and never cause drift.",
			remote: []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			cr:     newFeature(withDescription("A feature"), withTags("a", "b")),
			want:   managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
		},
		"MatchingRulesUpToDate": {
			reason: "An identical rule list is up to date.",
			remote: []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			cr:     newFeature(withDescription("A feature"), withTags("a", "b"), withRules(forceRule())),
			want:   managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
		},
		"DifferentRulesDrift": {
			reason: "A changed rule list needs an update.",
			remote: []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			cr:     newFeature(withDescription("A feature"), withTags("a", "b"), withRules(rolloutRule())),
			want:   managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := remote()
			r.Rules = tc.remote
			client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return r, nil }}
			e := external{client: client}
			got, err := e.Observe(context.Background(), tc.cr)
			if err != nil {
				t.Fatalf("\n%s\ne.Observe(...): unexpected error: %v", tc.reason, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

// TestObserveRulesStatus checks that atProvider.rules is populated from
// the observed rules.
func TestObserveRulesStatus(t *testing.T) {
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) {
		r := remote()
		r.Rules = []growthbook.FeatureRule{remoteRuleFrom(forceRule())}
		return r, nil
	}}
	cr := newFeature(withDescription("A feature"), withTags("a", "b"))

	e := external{client: client}
	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatalf("e.Observe(...): unexpected error %v", err)
	}
	if len(cr.Status.AtProvider.Rules) != 1 {
		t.Fatalf("atProvider.rules = %+v, want 1 entry", cr.Status.AtProvider.Rules)
	}
	if cr.Status.AtProvider.Rules[0].Type != "force" {
		t.Errorf("atProvider.rules[0].type = %q, want %q", cr.Status.AtProvider.Rules[0].Type, "force")
	}
}

// TestCreateRules checks that a non-nil rules list is sent on create.
func TestCreateRules(t *testing.T) {
	var gotReq growthbook.FeatureRequest
	client := &fakeClient{create: func(_ context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
		gotReq = req
		return &growthbook.Feature{ID: req.ID, ValueType: req.ValueType, DefaultValue: req.DefaultValue}, nil
	}}
	cr := newFeature(withRules(forceRule()))

	e := external{client: client}
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("e.Create(...): unexpected error %v", err)
	}
	if gotReq.Rules == nil {
		t.Fatalf("Create must send a non-nil rules pointer when spec.rules is set")
	}
	if len(*gotReq.Rules) != 1 {
		t.Errorf("Create sent %d rules, want 1", len(*gotReq.Rules))
	}
}

// TestCreateNilRules checks that a nil rules list leaves the field unset.
func TestCreateNilRules(t *testing.T) {
	var gotReq growthbook.FeatureRequest
	client := &fakeClient{create: func(_ context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
		gotReq = req
		return &growthbook.Feature{ID: req.ID, ValueType: req.ValueType, DefaultValue: req.DefaultValue}, nil
	}}
	cr := newFeature()

	e := external{client: client}
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("e.Create(...): unexpected error %v", err)
	}
	if gotReq.Rules != nil {
		t.Errorf("Create sent rules = %+v, want nil pointer for unmanaged rules", gotReq.Rules)
	}
}

// TestUpdateRules checks that Update sends the whole desired array,
// including an explicit empty one to clear rules.
func TestUpdateRules(t *testing.T) {
	cases := map[string]struct {
		reason string
		mod    func(*v1alpha1.Feature)
		check  func(t *testing.T, req growthbook.FeatureRequest)
	}{
		"SendsWholeArray": {
			reason: "Update replaces the whole rules array with the desired one.",
			mod:    withRules(forceRule(), rolloutRule()),
			check: func(t *testing.T, req growthbook.FeatureRequest) {
				if req.Rules == nil || len(*req.Rules) != 2 {
					t.Fatalf("Update rules = %+v, want 2 entries", req.Rules)
				}
			},
		},
		"EmptyClears": {
			reason: "An explicit empty rules list clears the feature's rules.",
			mod:    withEmptyRules(),
			check: func(t *testing.T, req growthbook.FeatureRequest) {
				if req.Rules == nil || len(*req.Rules) != 0 {
					t.Fatalf("Update rules = %+v, want a non-nil empty slice", req.Rules)
				}
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var gotReq growthbook.FeatureRequest
			client := &fakeClient{update: func(_ context.Context, _ string, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
				gotReq = req
				return remote(), nil
			}}
			cr := newFeature(tc.mod)

			e := external{client: client}
			if _, err := e.Update(context.Background(), cr); err != nil {
				t.Fatalf("\n%s\ne.Update(...): unexpected error: %v", tc.reason, err)
			}
			tc.check(t, gotReq)
		})
	}
}
