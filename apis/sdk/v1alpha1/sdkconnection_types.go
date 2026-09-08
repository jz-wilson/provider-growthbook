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

// SDKConnectionParameters are the configurable fields of a SDKConnection.
type SDKConnectionParameters struct {
	ConfigurableField string `json:"configurableField"`
}

// SDKConnectionObservation are the observable fields of a SDKConnection.
type SDKConnectionObservation struct {
	ConfigurableField string `json:"configurableField"`
	ObservableField   string `json:"observableField,omitempty"`
}

// A SDKConnectionSpec defines the desired state of a SDKConnection.
type SDKConnectionSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              SDKConnectionParameters `json:"forProvider"`
}

// A SDKConnectionStatus represents the observed state of a SDKConnection.
type SDKConnectionStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 SDKConnectionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A SDKConnection is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,growthbook}
type SDKConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SDKConnectionSpec   `json:"spec"`
	Status SDKConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SDKConnectionList contains a list of SDKConnection
type SDKConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SDKConnection `json:"items"`
}

// SDKConnection type metadata.
var (
	SDKConnectionKind             = reflect.TypeOf(SDKConnection{}).Name()
	SDKConnectionGroupKind        = schema.GroupKind{Group: Group, Kind: SDKConnectionKind}.String()
	SDKConnectionKindAPIVersion   = SDKConnectionKind + "." + SchemeGroupVersion.String()
	SDKConnectionGroupVersionKind = SchemeGroupVersion.WithKind(SDKConnectionKind)
)

func init() {
	SchemeBuilder.Register(&SDKConnection{}, &SDKConnectionList{})
}
