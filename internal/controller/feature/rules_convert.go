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
	"strconv"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// rulesRequest converts the desired rules into the API request pointer. A
// nil slice keeps rules unmanaged (the pointer stays nil, so the field is
// omitted and GrowthBook leaves the feature's rules untouched); a non-nil
// slice, including an empty one, is authoritative and replaces the whole
// rules array.
func rulesRequest(rules []v1alpha1.FeatureRule) (*[]growthbook.FeatureRule, error) {
	if rules == nil {
		return nil, nil
	}
	out := make([]growthbook.FeatureRule, len(rules))
	for i, r := range rules {
		gr, err := ruleRequest(r)
		if err != nil {
			return nil, errors.Wrapf(err, "rule %d", i)
		}
		out[i] = gr
	}
	return &out, nil
}

// ruleRequest converts one desired rule. Coverage arrives on the CRD as a
// decimal string and travels on the wire as a JSON number; the CRD's
// pattern validation should prevent a non-decimal value, but this is
// defensive about it the same way Project settings are.
func ruleRequest(r v1alpha1.FeatureRule) (growthbook.FeatureRule, error) {
	gr := growthbook.FeatureRule{
		Type:            r.Type,
		ID:              derefString(r.ID),
		Description:     derefString(r.Description),
		Enabled:         r.Enabled,
		Condition:       derefString(r.Condition),
		SavedGroups:     savedGroupsRequest(r.SavedGroups),
		AllEnvironments: derefBool(r.AllEnvironments),
		Environments:    r.Environments,
		Value:           derefString(r.Value),
		HashAttribute:   derefString(r.HashAttribute),
		ExperimentID:    derefString(r.ExperimentID),
		Variations:      variationsRequest(r.Variations),
	}

	if r.Coverage != nil {
		f, err := strconv.ParseFloat(*r.Coverage, 64)
		if err != nil {
			return growthbook.FeatureRule{}, errors.Wrap(err, "coverage is not a decimal")
		}
		gr.Coverage = &f
	}

	return gr, nil
}

func savedGroupsRequest(in []v1alpha1.SavedGroupTargeting) []growthbook.FeatureSavedGroupTargeting {
	if in == nil {
		return nil
	}
	out := make([]growthbook.FeatureSavedGroupTargeting, len(in))
	for i, sg := range in {
		out[i] = growthbook.FeatureSavedGroupTargeting{Match: sg.Match, IDs: sg.IDs}
	}
	return out
}

func variationsRequest(in []v1alpha1.FeatureRuleVariation) []growthbook.FeatureRuleVariation {
	if in == nil {
		return nil
	}
	out := make([]growthbook.FeatureRuleVariation, len(in))
	for i, v := range in {
		out[i] = growthbook.FeatureRuleVariation{VariationID: v.VariationID, Value: v.Value}
	}
	return out
}

// rulesObservation maps the API's observed rules onto status.atProvider.
func rulesObservation(rules []growthbook.FeatureRule) []v1alpha1.FeatureRuleObservation {
	if rules == nil {
		return nil
	}
	out := make([]v1alpha1.FeatureRuleObservation, len(rules))
	for i, r := range rules {
		out[i] = v1alpha1.FeatureRuleObservation{
			ID:           r.ID,
			Type:         r.Type,
			Enabled:      enabledOrTrue(r.Enabled),
			Environments: r.Environments,
		}
	}
	return out
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	return b != nil && *b
}

// enabledOrTrue treats an unset Enabled flag as on, matching GrowthBook's
// default for a rule.
func enabledOrTrue(b *bool) bool {
	return b == nil || *b
}
