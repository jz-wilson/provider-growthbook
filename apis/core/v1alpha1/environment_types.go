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

// EnvironmentParameters are the configurable fields of a GrowthBook
// Environment. The environment id is the resource's external name
// (crossplane.io/external-name annotation), defaulting to metadata.name.
type EnvironmentParameters struct {
	// Description is free text shown in the GrowthBook UI.
	// +optional
	Description *string `json:"description,omitempty"`

	// ToggleOnList shows this environment's toggle on the feature list page.
	// +optional
	ToggleOnList *bool `json:"toggleOnList,omitempty"`

	// DefaultState is the initial on/off state new features get in this
	// environment.
	// +optional
	DefaultState *bool `json:"defaultState,omitempty"`

	// Projects restricts the environment to these project ids. Empty means
	// all projects. Compared as a set.
	// +optional
	Projects []string `json:"projects,omitempty"`

	// Parent is an environment id to inherit feature rules from. Create-only
	// and requires a GrowthBook Enterprise license.
	// +optional
	// +immutable
	Parent *string `json:"parent,omitempty"`
}

// EnvironmentObservation are the observable fields of a GrowthBook Environment.
type EnvironmentObservation struct {
	// ID is the environment id as stored by GrowthBook.
	ID string `json:"id,omitempty"`

	// Parent is the inherited-from environment, if any.
	Parent string `json:"parent,omitempty"`
}

// An EnvironmentSpec defines the desired state of an Environment.
type EnvironmentSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              EnvironmentParameters `json:"forProvider"`
}

// An EnvironmentStatus represents the observed state of an Environment.
type EnvironmentStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 EnvironmentObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An Environment is a GrowthBook environment such as production or staging.
// Feature rules are evaluated per environment.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,growthbook}
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvironmentList contains a list of Environment
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

// Environment type metadata.
var (
	EnvironmentKind             = reflect.TypeOf(Environment{}).Name()
	EnvironmentGroupKind        = schema.GroupKind{Group: Group, Kind: EnvironmentKind}.String()
	EnvironmentKindAPIVersion   = EnvironmentKind + "." + SchemeGroupVersion.String()
	EnvironmentGroupVersionKind = SchemeGroupVersion.WithKind(EnvironmentKind)
)

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}
