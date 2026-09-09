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

package v1alpha1

// SavedGroupTargeting scopes a rule to users in (or not in) a set of saved
// groups.
type SavedGroupTargeting struct {
	// Match selects how the listed saved groups are combined.
	// +kubebuilder:validation:Enum=all;none;any
	Match string `json:"match"`

	// IDs are the saved group ids this targeting entry references.
	IDs []string `json:"ids"`
}

// FeatureRuleVariation maps one experiment variation to the value this
// feature serves when that variation is assigned.
type FeatureRuleVariation struct {
	// VariationID is the id of the experiment variation.
	VariationID string `json:"variationId"`

	// Value is the feature value served for this variation.
	Value string `json:"value"`
}

// FeatureRule is one entry in a Feature's rules array. Rules are evaluated
// in list order; the first matching rule wins. Type-specific fields are
// only meaningful for the matching Type and are otherwise ignored.
//
// Safe-rollout rules are out of scope: they only round-trip via a
// server-assigned safeRolloutId this provider does not model, so they
// cannot be authored or reconciled here.
type FeatureRule struct {
	// Type selects the rule kind.
	// +kubebuilder:validation:Enum=force;rollout;experiment-ref
	Type string `json:"type"`

	// ID is the server-assigned rule id. Leave empty to let GrowthBook
	// assign one on create; set it (as observed via status.atProvider) to
	// keep updates stable and target a specific existing rule.
	// +optional
	ID *string `json:"id,omitempty"`

	// Description is free text shown in the GrowthBook UI.
	// +optional
	Description *string `json:"description,omitempty"`

	// Enabled toggles this rule on or off.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Condition is a JSON-encoded targeting condition, passed through to
	// GrowthBook verbatim.
	// +optional
	Condition *string `json:"condition,omitempty"`

	// SavedGroups scopes this rule to users in (or not in) saved groups.
	// +optional
	SavedGroups []SavedGroupTargeting `json:"savedGroups,omitempty"`

	// AllEnvironments, when true, applies this rule in every environment.
	// When false, only the environments listed in Environments receive
	// the rule.
	// +optional
	AllEnvironments *bool `json:"allEnvironments,omitempty"`

	// Environments lists the environment ids this rule applies to. Used
	// only when AllEnvironments is false.
	// +optional
	Environments []string `json:"environments,omitempty"`

	// Value is the value served when this rule matches. Applies to the
	// "force" and "rollout" types.
	// +optional
	Value *string `json:"value,omitempty"`

	// Coverage is the percentage of matched users to include, as a
	// decimal string between "0" and "1" inclusive (for example "0.25").
	// Applies to the "rollout" type; required when the rollout should
	// serve fewer than 100% of matched users.
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`
	Coverage *string `json:"coverage,omitempty"`

	// HashAttribute is the attribute to hash on for consistent user
	// assignment. Applies to (and required by GrowthBook for) the
	// "rollout" type.
	// +optional
	HashAttribute *string `json:"hashAttribute,omitempty"`

	// ExperimentID links this rule to an existing experiment. Applies to
	// the "experiment-ref" type.
	// +optional
	ExperimentID *string `json:"experimentId,omitempty"`

	// Variations maps experiment variations to the values this feature
	// serves for each. Applies to the "experiment-ref" type.
	// +optional
	Variations []FeatureRuleVariation `json:"variations,omitempty"`
}

// FeatureRuleObservation reports the observed state of one rule on a
// Feature.
type FeatureRuleObservation struct {
	// ID is the server-assigned rule id.
	ID string `json:"id,omitempty"`

	// Type is the observed rule kind.
	Type string `json:"type,omitempty"`

	// Enabled reports whether the rule is on.
	Enabled bool `json:"enabled,omitempty"`

	// Environments reports the environment ids the rule is scoped to.
	Environments []string `json:"environments,omitempty"`
}
