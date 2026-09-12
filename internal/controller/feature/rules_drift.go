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

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// rulesUpToDate compares the desired rules against the observed ones. Rules
// are opt-in: a nil desired slice means rules are unmanaged and never
// drift, regardless of what GrowthBook reports. A non-nil slice compares
// ordered: a reorder, an added or removed rule, or a changed field on any
// rule counts as drift.
func rulesUpToDate(want []v1alpha1.FeatureRule, got []growthbook.FeatureRule) bool {
	if want == nil {
		return true
	}
	if len(want) != len(got) {
		return false
	}
	for i := range want {
		if !ruleUpToDate(want[i], got[i]) {
			return false
		}
	}
	return true
}

// ruleUpToDate compares one rule pair. The server-assigned id is ignored
// unless the user set one, in which case it must match.
func ruleUpToDate(w v1alpha1.FeatureRule, g growthbook.FeatureRule) bool {
	switch {
	case w.Type != g.Type:
		return false
	case w.ID != nil && *w.ID != g.ID:
		return false
	case enabledOrTrue(w.Enabled) != enabledOrTrue(g.Enabled):
		return false
	case derefString(w.Condition) != g.Condition:
		return false
	case !savedGroupsUpToDate(w.SavedGroups, g.SavedGroups):
		return false
	case !environmentsRuleUpToDate(w, g):
		return false
	}
	return ruleValueUpToDate(w, g)
}

// environmentsRuleUpToDate compares allEnvironments/environments with set
// semantics. Environments is only meaningful when allEnvironments is
// false.
func environmentsRuleUpToDate(w v1alpha1.FeatureRule, g growthbook.FeatureRule) bool {
	wantAll := derefBool(w.AllEnvironments)
	if wantAll != g.AllEnvironments {
		return false
	}
	if wantAll {
		return true
	}
	return sameSet(w.Environments, g.Environments)
}

// ruleValueUpToDate compares the type-specific fields: value and coverage
// for "rollout", value alone for "force", experimentId and variations for
// "experiment-ref". Fields irrelevant to a rule's type are zero on both
// sides, so comparing all of them unconditionally is safe.
func ruleValueUpToDate(w v1alpha1.FeatureRule, g growthbook.FeatureRule) bool {
	switch {
	case derefString(w.Value) != g.Value:
		return false
	case derefString(w.HashAttribute) != g.HashAttribute:
		return false
	case derefString(w.ExperimentID) != g.ExperimentID:
		return false
	case !coverageUpToDate(w.Coverage, g.Coverage):
		return false
	}
	return variationsUpToDate(w.Variations, g.Variations)
}

// coverageUpToDate compares coverage numerically. An unparseable desired
// value counts as out of date, the same way Project settings treat it, so
// Update surfaces the real parse error.
func coverageUpToDate(w *string, g *float64) bool {
	if w == nil {
		return g == nil
	}
	f, err := strconv.ParseFloat(*w, 64)
	if err != nil {
		return false
	}
	return g != nil && f == *g
}

// savedGroupsUpToDate compares entries in order (each entry's match mode
// matters), but the ids within one entry compare as a set.
func savedGroupsUpToDate(w []v1alpha1.SavedGroupTargeting, g []growthbook.FeatureSavedGroupTargeting) bool {
	if len(w) != len(g) {
		return false
	}
	for i := range w {
		if w[i].Match != g[i].Match {
			return false
		}
		if !sameSet(w[i].IDs, g[i].IDs) {
			return false
		}
	}
	return true
}

// variationsUpToDate compares experiment-ref variations by variationId
// rather than position.
func variationsUpToDate(w []v1alpha1.FeatureRuleVariation, g []growthbook.FeatureRuleVariation) bool {
	if len(w) != len(g) {
		return false
	}
	gm := make(map[string]string, len(g))
	for _, v := range g {
		gm[v.VariationID] = v.Value
	}
	for _, v := range w {
		gv, ok := gm[v.VariationID]
		if !ok || gv != v.Value {
			return false
		}
	}
	return true
}
