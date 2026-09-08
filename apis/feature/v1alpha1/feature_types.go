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

// FeatureParameters are the configurable fields of a Feature.
type FeatureParameters struct {
	ConfigurableField string `json:"configurableField"`
}

// FeatureObservation are the observable fields of a Feature.
type FeatureObservation struct {
	ConfigurableField string `json:"configurableField"`
	ObservableField   string `json:"observableField,omitempty"`
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

// A Feature is an example API type.
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
