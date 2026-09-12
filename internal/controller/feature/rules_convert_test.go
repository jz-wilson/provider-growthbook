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
	"testing"

	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

func TestRulesRequest(t *testing.T) {
	cases := map[string]struct {
		reason  string
		rules   []v1alpha1.FeatureRule
		want    *[]growthbook.FeatureRule
		wantErr bool
	}{
		"Nil": {
			reason: "A nil rules slice must produce a nil pointer so the request omits the field and GrowthBook leaves rules unchanged.",
			rules:  nil,
			want:   nil,
		},
		"Empty": {
			reason: "A non-nil empty slice must produce a non-nil pointer to an empty slice, which clears the feature's rules.",
			rules:  []v1alpha1.FeatureRule{},
			want:   &[]growthbook.FeatureRule{},
		},
		"Force": {
			reason: "A force rule converts its condition, savedGroups, environments, and value.",
			rules: []v1alpha1.FeatureRule{{
				Type:            ruleTypeForce,
				Description:     ptr("force rule"),
				Enabled:         ptr(true),
				Condition:       ptr(`{"country":"US"}`),
				SavedGroups:     []v1alpha1.SavedGroupTargeting{{Match: matchAny, IDs: []string{savedGroupID1, "sg_2"}}},
				AllEnvironments: ptr(false),
				Environments:    []string{envProduction},
				Value:           ptr(valTrue),
			}},
			want: &[]growthbook.FeatureRule{{
				Type:            ruleTypeForce,
				Description:     "force rule",
				Enabled:         ptr(true),
				Condition:       `{"country":"US"}`,
				SavedGroups:     []growthbook.FeatureSavedGroupTargeting{{Match: matchAny, IDs: []string{savedGroupID1, "sg_2"}}},
				AllEnvironments: false,
				Environments:    []string{envProduction},
				Value:           valTrue,
			}},
		},
		"Rollout": {
			reason: "A rollout rule converts its coverage string to a wire float and its hashAttribute.",
			rules: []v1alpha1.FeatureRule{{
				Type:            ruleTypeRollout,
				AllEnvironments: ptr(true),
				Value:           ptr(valTrue),
				Coverage:        ptr("0.25"),
				HashAttribute:   ptr("id"),
			}},
			want: &[]growthbook.FeatureRule{{
				Type:            ruleTypeRollout,
				AllEnvironments: true,
				Value:           valTrue,
				Coverage:        ptr(0.25),
				HashAttribute:   "id",
			}},
		},
		"ExperimentRef": {
			reason: "An experiment-ref rule converts its experimentId and variations.",
			rules: []v1alpha1.FeatureRule{{
				Type:            "experiment-ref",
				ID:              ptr("rule_1"),
				AllEnvironments: ptr(true),
				ExperimentID:    ptr("exp_1"),
				Variations: []v1alpha1.FeatureRuleVariation{
					{VariationID: "0", Value: valFalse},
					{VariationID: "1", Value: valTrue},
				},
			}},
			want: &[]growthbook.FeatureRule{{
				Type:            "experiment-ref",
				ID:              "rule_1",
				AllEnvironments: true,
				ExperimentID:    "exp_1",
				Variations: []growthbook.FeatureRuleVariation{
					{VariationID: "0", Value: valFalse},
					{VariationID: "1", Value: valTrue},
				},
			}},
		},
		"BadCoverage": {
			reason: "A non-decimal coverage value is a defensive error, even though the CRD pattern should prevent it.",
			rules: []v1alpha1.FeatureRule{{
				Type:            ruleTypeRollout,
				AllEnvironments: ptr(true),
				Coverage:        ptr("not-a-number"),
			}},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := rulesRequest(tc.rules)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("\n%s\nrulesRequest(...): expected an error, got none", tc.reason)
				}
				return
			}
			if err != nil {
				t.Fatalf("\n%s\nrulesRequest(...): unexpected error: %v", tc.reason, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nrulesRequest(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestRulesObservation(t *testing.T) {
	got := rulesObservation([]growthbook.FeatureRule{
		{ID: "rule_1", Type: ruleTypeForce, Enabled: ptr(true), Environments: []string{envProduction}},
		{ID: "rule_2", Type: ruleTypeRollout, Enabled: nil, AllEnvironments: true},
	})
	want := []v1alpha1.FeatureRuleObservation{
		{ID: "rule_1", Type: ruleTypeForce, Enabled: true, Environments: []string{envProduction}},
		{ID: "rule_2", Type: ruleTypeRollout, Enabled: true},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rulesObservation(...): -want, +got:\n%s\n", diff)
	}
}

func TestRulesObservationNil(t *testing.T) {
	if got := rulesObservation(nil); got != nil {
		t.Errorf("rulesObservation(nil) = %+v, want nil", got)
	}
}
