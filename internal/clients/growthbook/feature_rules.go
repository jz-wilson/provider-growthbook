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

// FeatureSavedGroupTargeting scopes a rule to users in (or not in) a set
// of saved groups, as returned or sent by the v2 API.
type FeatureSavedGroupTargeting struct {
	Match string   `json:"match"`
	IDs   []string `json:"ids"`
}

// FeatureRuleVariation maps one experiment variation to the value an
// experiment-ref rule serves for it.
type FeatureRuleVariation struct {
	VariationID string `json:"variationId,omitempty"`
	Value       string `json:"value"`
}

// FeatureRule is one entry in a feature's flat v2 rules array. It is a
// single JSON-faithful struct across all rule kinds: Type discriminates
// which of the type-specific fields apply, and every other field is
// omitempty so a given rule only ever emits the properties relevant to
// its kind. Coverage travels on the wire as a JSON number (GrowthBook's
// schema declares it a float between 0 and 1); the CRD layer represents
// it as a decimal string and the controller converts between the two.
//
// Safe-rollout rules are intentionally not modeled: they only round-trip
// via a server-assigned safeRolloutId this package does not decode, so a
// safe-rollout rule present on a feature passes through GetFeature's
// response with Type "safe-rollout" and its type-specific fields empty.
type FeatureRule struct {
	Type            string                       `json:"type"`
	ID              string                       `json:"id,omitempty"`
	Description     string                       `json:"description,omitempty"`
	Enabled         *bool                        `json:"enabled,omitempty"`
	Condition       string                       `json:"condition,omitempty"`
	SavedGroups     []FeatureSavedGroupTargeting `json:"savedGroups,omitempty"`
	AllEnvironments bool                         `json:"allEnvironments"`
	Environments    []string                     `json:"environments,omitempty"`

	// Value, Coverage, and HashAttribute apply to the "force" and
	// "rollout" types.
	Value         string   `json:"value,omitempty"`
	Coverage      *float64 `json:"coverage,omitempty"`
	HashAttribute string   `json:"hashAttribute,omitempty"`

	// ExperimentID and Variations apply to the "experiment-ref" type.
	ExperimentID string                 `json:"experimentId,omitempty"`
	Variations   []FeatureRuleVariation `json:"variations,omitempty"`
}
