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

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

func forceRule() v1alpha1.FeatureRule {
	return v1alpha1.FeatureRule{
		Type:            ruleTypeForce,
		AllEnvironments: ptr(true),
		Condition:       ptr(`{"country":"US"}`),
		Value:           ptr(valTrue),
	}
}

func rolloutRule() v1alpha1.FeatureRule {
	return v1alpha1.FeatureRule{
		Type:            ruleTypeRollout,
		AllEnvironments: ptr(true),
		Value:           ptr(valTrue),
		Coverage:        ptr("0.50"),
		HashAttribute:   ptr("id"),
	}
}

func remoteRuleFrom(r v1alpha1.FeatureRule) growthbook.FeatureRule {
	gr, err := ruleRequest(r)
	if err != nil {
		panic(err)
	}
	if gr.Enabled == nil {
		gr.Enabled = ptr(true)
	}
	return gr
}

func TestRulesUpToDate(t *testing.T) {
	cases := map[string]struct {
		reason   string
		want     []v1alpha1.FeatureRule
		got      []growthbook.FeatureRule
		upToDate bool
	}{
		"NilNeverDrifts": {
			reason:   "Rules left nil are unmanaged and never drift, however different the remote rules are.",
			want:     nil,
			got:      []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			upToDate: true,
		},
		"EmptyMatchesEmpty": {
			reason:   "An explicit empty desired list matches an empty remote list.",
			want:     []v1alpha1.FeatureRule{},
			got:      []growthbook.FeatureRule{},
			upToDate: true,
		},
		"EmptyDriftsAgainstNonEmpty": {
			reason:   "An explicit empty desired list drifts against a non-empty remote list.",
			want:     []v1alpha1.FeatureRule{},
			got:      []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			upToDate: false,
		},
		"SameRuleMatches": {
			reason:   "An identical single rule is up to date.",
			want:     []v1alpha1.FeatureRule{forceRule()},
			got:      []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			upToDate: true,
		},
		"ReorderDrifts": {
			reason:   "Rules are compared in order, so a reorder is drift even though the set of rules is unchanged.",
			want:     []v1alpha1.FeatureRule{forceRule(), rolloutRule()},
			got:      []growthbook.FeatureRule{remoteRuleFrom(rolloutRule()), remoteRuleFrom(forceRule())},
			upToDate: false,
		},
		"ChangedConditionDrifts": {
			reason: "A changed condition on a rule is drift.",
			want:   []v1alpha1.FeatureRule{forceRule()},
			got: func() []growthbook.FeatureRule {
				r := remoteRuleFrom(forceRule())
				r.Condition = `{"country":"CA"}`
				return []growthbook.FeatureRule{r}
			}(),
			upToDate: false,
		},
		"CoverageDecimalEquivalentNoDrift": {
			reason:   `"0.50" and 0.5 are numerically equal, so they must not drift.`,
			want:     []v1alpha1.FeatureRule{rolloutRule()},
			got:      []growthbook.FeatureRule{remoteRuleFrom(rolloutRule())},
			upToDate: true,
		},
		"CoverageChangedDrifts": {
			reason: "A numerically different coverage is drift.",
			want:   []v1alpha1.FeatureRule{rolloutRule()},
			got: func() []growthbook.FeatureRule {
				r := remoteRuleFrom(rolloutRule())
				r.Coverage = ptr(0.75)
				return []growthbook.FeatureRule{r}
			}(),
			upToDate: false,
		},
		"AllEnvironmentsTrueIgnoresEnvironmentsList": {
			reason: "When allEnvironments is true, the environments list is irrelevant on both sides.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: ptr(true), Value: ptr(valTrue),
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: true, Value: valTrue, Enabled: ptr(true), Environments: []string{envStaging},
			}},
			upToDate: true,
		},
		"EnvironmentsSetOrderInsensitive": {
			reason: "The environments list compares as a set, so reordered entries do not drift.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: ptr(false), Environments: []string{envProduction, envStaging}, Value: ptr(valTrue),
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: false, Environments: []string{envStaging, envProduction}, Value: valTrue, Enabled: ptr(true),
			}},
			upToDate: true,
		},
		"EnvironmentsSetDrift": {
			reason: "A different set of environments drifts.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: ptr(false), Environments: []string{envProduction}, Value: ptr(valTrue),
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: false, Environments: []string{envStaging}, Value: valTrue, Enabled: ptr(true),
			}},
			upToDate: false,
		},
		"SavedGroupsIDsOrderInsensitive": {
			reason: "The ids inside one savedGroups entry compare as a set.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: ptr(true), Value: ptr(valTrue),
				SavedGroups: []v1alpha1.SavedGroupTargeting{{Match: matchAny, IDs: []string{savedGroupID1, "sg_2"}}},
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: true, Value: valTrue, Enabled: ptr(true),
				SavedGroups: []growthbook.FeatureSavedGroupTargeting{{Match: matchAny, IDs: []string{"sg_2", savedGroupID1}}},
			}},
			upToDate: true,
		},
		"SavedGroupsMatchDrifts": {
			reason: "A changed match mode is drift even if the id set is identical.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: ptr(true), Value: ptr(valTrue),
				SavedGroups: []v1alpha1.SavedGroupTargeting{{Match: matchAny, IDs: []string{savedGroupID1}}},
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: true, Value: valTrue, Enabled: ptr(true),
				SavedGroups: []growthbook.FeatureSavedGroupTargeting{{Match: "none", IDs: []string{savedGroupID1}}},
			}},
			upToDate: false,
		},
		"VariationsByIDOrderInsensitive": {
			reason: "Experiment-ref variations compare by variationId, not position.",
			want: []v1alpha1.FeatureRule{{
				Type: "experiment-ref", AllEnvironments: ptr(true), ExperimentID: ptr("exp_1"),
				Variations: []v1alpha1.FeatureRuleVariation{
					{VariationID: "1", Value: valTrue},
					{VariationID: "0", Value: "false"},
				},
			}},
			got: []growthbook.FeatureRule{{
				Type: "experiment-ref", AllEnvironments: true, ExperimentID: "exp_1", Enabled: ptr(true),
				Variations: []growthbook.FeatureRuleVariation{
					{VariationID: "0", Value: "false"},
					{VariationID: "1", Value: valTrue},
				},
			}},
			upToDate: true,
		},
		"UserSetIDMustMatch": {
			reason: "A user-set rule id must match the observed id.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, ID: ptr("rule_1"), AllEnvironments: ptr(true), Value: ptr(valTrue),
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, ID: "rule_2", AllEnvironments: true, Value: valTrue, Enabled: ptr(true),
			}},
			upToDate: false,
		},
		"ServerAssignedIDIgnoredWhenUnset": {
			reason: "A server-assigned id the user never set does not cause drift.",
			want: []v1alpha1.FeatureRule{{
				Type: ruleTypeForce, AllEnvironments: ptr(true), Value: ptr(valTrue),
			}},
			got: []growthbook.FeatureRule{{
				Type: ruleTypeForce, ID: "rule_1", AllEnvironments: true, Value: valTrue, Enabled: ptr(true),
			}},
			upToDate: true,
		},
		"LengthMismatchDrifts": {
			reason:   "A different number of rules is always drift.",
			want:     []v1alpha1.FeatureRule{forceRule(), rolloutRule()},
			got:      []growthbook.FeatureRule{remoteRuleFrom(forceRule())},
			upToDate: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := rulesUpToDate(tc.want, tc.got); got != tc.upToDate {
				t.Errorf("\n%s\nrulesUpToDate(...) = %v, want %v", tc.reason, got, tc.upToDate)
			}
		})
	}
}
