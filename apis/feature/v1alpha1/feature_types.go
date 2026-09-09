/*
Copyright 2025 The Crossplane Authors.

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

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// FeatureEnvironment is the desired state of a Feature within one
// environment.
type FeatureEnvironment struct {
	// Enabled toggles the feature on or off in this environment.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// FeatureParameters are the configurable fields of a GrowthBook Feature.
// The feature key is the resource's external name (crossplane.io/
// external-name annotation), defaulting to metadata.name.
//
// This milestone deliberately covers only the feature's value and its
// per-environment enabled toggles. Rules, prerequisites, revisions, and
// JSON schema validation are not modeled and are left for a later
// milestone; a Feature managed here will report ResourceUpToDate based
// solely on the fields below even if it also carries rules configured
// out of band (for example through the GrowthBook UI).
type FeatureParameters struct {
	// ValueType is the data type of the feature payload. Immutable after
	// creation.
	// +kubebuilder:validation:Enum=boolean;string;number;json
	ValueType string `json:"valueType"`

	// DefaultValue is the value served when the feature is enabled. Its
	// type must match ValueType; for "number" and "json" the literal is
	// passed through as a string exactly as GrowthBook expects it.
	DefaultValue string `json:"defaultValue"`

	// Description is free text shown in the GrowthBook UI.
	// +optional
	Description *string `json:"description,omitempty"`

	// Project restricts the feature to one project id.
	// +optional
	Project *string `json:"project,omitempty"`

	// Tags are labels associated with the feature. Nil means unmanaged;
	// a set list compares and replaces as a set.
	// +optional
	Tags []string `json:"tags,omitempty"`

	// Archived hides the feature from the default feature list.
	// +optional
	Archived *bool `json:"archived,omitempty"`

	// Owner is the userId or email address of the feature's owner.
	// +optional
	Owner *string `json:"owner,omitempty"`

	// Environments maps an environment id to its desired enabled state.
	// Only the environments present here are managed; environments left
	// out are never touched.
	// +optional
	Environments map[string]FeatureEnvironment `json:"environments,omitempty"`
}

// FeatureEnvironmentObservation is the observed state of a Feature within
// one environment.
type FeatureEnvironmentObservation struct {
	// Enabled reports whether the feature is on in this environment.
	Enabled bool `json:"enabled,omitempty"`
}

// FeatureRevisionObservation reports the feature's current published
// revision.
type FeatureRevisionObservation struct {
	// Version is the revision number.
	Version int `json:"version,omitempty"`
}

// FeatureObservation are the observable fields of a GrowthBook Feature.
type FeatureObservation struct {
	// ID is the feature key as stored by GrowthBook.
	ID string `json:"id,omitempty"`

	// Revision is the feature's current published revision.
	Revision FeatureRevisionObservation `json:"revision,omitempty"`

	// DateCreated is when GrowthBook created the feature.
	DateCreated string `json:"dateCreated,omitempty"`

	// DateUpdated is when GrowthBook last updated the feature.
	DateUpdated string `json:"dateUpdated,omitempty"`

	// Environments reports the observed enabled state per environment.
	Environments map[string]FeatureEnvironmentObservation `json:"environments,omitempty"`
}

// A FeatureSpec defines the desired state of a Feature.
type FeatureSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              FeatureParameters `json:"forProvider"`
}

// A FeatureStatus represents the observed state of a Feature.
type FeatureStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 FeatureObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Feature is a GrowthBook feature flag. This milestone manages its
// value and per-environment enabled toggles; rules are not modeled.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,growthbook}
type Feature struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FeatureSpec   `json:"spec"`
	Status FeatureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FeatureList contains a list of Feature
type FeatureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Feature `json:"items"`
}

// Feature type metadata.
var (
	FeatureKind             = reflect.TypeOf(Feature{}).Name()
	FeatureGroupKind        = schema.GroupKind{Group: Group, Kind: FeatureKind}.String()
	FeatureKindAPIVersion   = FeatureKind + "." + SchemeGroupVersion.String()
	FeatureGroupVersionKind = SchemeGroupVersion.WithKind(FeatureKind)
)

func init() {
	SchemeBuilder.Register(&Feature{}, &FeatureList{})
}
