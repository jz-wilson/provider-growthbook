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

// SDKConnectionParameters are the configurable fields of a GrowthBook SDK
// Connection. Every optional field is a pointer so that only fields the user
// actually sets participate in drift detection; a nil pointer means
// "unmanaged", not "false"/"empty".
type SDKConnectionParameters struct {
	// Name is the human-readable connection name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Language is the SDK language/platform this connection targets, for
	// example "javascript", "react", "go", or "nodejs".
	// +kubebuilder:validation:MinLength=1
	Language string `json:"language"`

	// Environment is the id of the GrowthBook environment this connection
	// serves.
	// +kubebuilder:validation:MinLength=1
	Environment string `json:"environment"`

	// Projects restricts the connection to the given project ids. A nil
	// value leaves project scoping unmanaged; an empty list clears it.
	// +optional
	Projects []string `json:"projects,omitempty"`

	// SDKVersion is the SDK version this connection targets.
	// +optional
	SDKVersion *string `json:"sdkVersion,omitempty"`

	// EncryptPayload enables payload encryption for this connection.
	// +optional
	EncryptPayload *bool `json:"encryptPayload,omitempty"`

	// IncludeVisualExperiments includes visual editor experiments in the
	// SDK payload.
	// +optional
	IncludeVisualExperiments *bool `json:"includeVisualExperiments,omitempty"`

	// IncludeDraftExperiments includes draft (not yet running) experiments
	// in the SDK payload.
	// +optional
	IncludeDraftExperiments *bool `json:"includeDraftExperiments,omitempty"`

	// IncludeDraftExperimentRefs includes experiment-ref rules linked to
	// draft experiments in the SDK payload.
	// +optional
	IncludeDraftExperimentRefs *bool `json:"includeDraftExperimentRefs,omitempty"`

	// IncludeExperimentNames includes experiment names in the SDK payload.
	// +optional
	IncludeExperimentNames *bool `json:"includeExperimentNames,omitempty"`

	// IncludeRedirectExperiments includes URL redirect experiments in the
	// SDK payload.
	// +optional
	IncludeRedirectExperiments *bool `json:"includeRedirectExperiments,omitempty"`

	// IncludeRuleIds includes rule ids in the SDK payload.
	// +optional
	IncludeRuleIds *bool `json:"includeRuleIds,omitempty"`

	// IncludeProjectIdInMetadata includes the project id in feature
	// metadata.
	// +optional
	IncludeProjectIdInMetadata *bool `json:"includeProjectIdInMetadata,omitempty"`

	// IncludeCustomFieldsInMetadata includes custom fields in feature
	// metadata.
	// +optional
	IncludeCustomFieldsInMetadata *bool `json:"includeCustomFieldsInMetadata,omitempty"`

	// AllowedCustomFieldsInMetadata limits which custom fields are included
	// when IncludeCustomFieldsInMetadata is set.
	// +optional
	AllowedCustomFieldsInMetadata []string `json:"allowedCustomFieldsInMetadata,omitempty"`

	// IncludeTagsInMetadata includes tags in feature metadata.
	// +optional
	IncludeTagsInMetadata *bool `json:"includeTagsInMetadata,omitempty"`

	// IncludeExperimentScheduleInMetadata includes experiment scheduling
	// info in feature metadata.
	// +optional
	IncludeExperimentScheduleInMetadata *bool `json:"includeExperimentScheduleInMetadata,omitempty"`

	// ProxyEnabled enables the GrowthBook proxy for this connection.
	// +optional
	ProxyEnabled *bool `json:"proxyEnabled,omitempty"`

	// ProxyHost is the proxy host to use when ProxyEnabled is true.
	// +optional
	ProxyHost *string `json:"proxyHost,omitempty"`

	// HashSecureAttributes hashes secure attributes before they leave the
	// server.
	// +optional
	HashSecureAttributes *bool `json:"hashSecureAttributes,omitempty"`

	// RemoteEvalEnabled enables remote evaluation for this connection.
	// +optional
	RemoteEvalEnabled *bool `json:"remoteEvalEnabled,omitempty"`

	// SavedGroupReferencesEnabled enables saved group references in the SDK
	// payload.
	// +optional
	SavedGroupReferencesEnabled *bool `json:"savedGroupReferencesEnabled,omitempty"`

	// IncludeReferencedPrerequisites carries prerequisite feature flags into
	// the payload even when they target other projects.
	// +optional
	IncludeReferencedPrerequisites *bool `json:"includeReferencedPrerequisites,omitempty"`
}

// SDKConnectionInitParameters are the fields of a GrowthBook SDK Connection
// that are applied only when the resource is created. Crossplane ignores
// later changes to these fields in the external resource; use them together
// with managementPolicies that omit LateInitialize to avoid conflicts with
// forProvider. Any field also set in forProvider is overridden by
// forProvider.
type SDKConnectionInitParameters struct {
	// Name is the human-readable connection name.
	// +optional
	Name *string `json:"name,omitempty"`

	// Language is the SDK language/platform this connection targets, for
	// example "javascript", "react", "go", or "nodejs".
	// +optional
	Language *string `json:"language,omitempty"`

	// Environment is the id of the GrowthBook environment this connection
	// serves.
	// +optional
	Environment *string `json:"environment,omitempty"`

	// Projects restricts the connection to the given project ids.
	// +optional
	Projects []string `json:"projects,omitempty"`

	// SDKVersion is the SDK version this connection targets.
	// +optional
	SDKVersion *string `json:"sdkVersion,omitempty"`

	// EncryptPayload enables payload encryption for this connection.
	// +optional
	EncryptPayload *bool `json:"encryptPayload,omitempty"`

	// IncludeVisualExperiments includes visual editor experiments in the
	// SDK payload.
	// +optional
	IncludeVisualExperiments *bool `json:"includeVisualExperiments,omitempty"`

	// IncludeDraftExperiments includes draft (not yet running) experiments
	// in the SDK payload.
	// +optional
	IncludeDraftExperiments *bool `json:"includeDraftExperiments,omitempty"`

	// IncludeDraftExperimentRefs includes experiment-ref rules linked to
	// draft experiments in the SDK payload.
	// +optional
	IncludeDraftExperimentRefs *bool `json:"includeDraftExperimentRefs,omitempty"`

	// IncludeExperimentNames includes experiment names in the SDK payload.
	// +optional
	IncludeExperimentNames *bool `json:"includeExperimentNames,omitempty"`

	// IncludeRedirectExperiments includes URL redirect experiments in the
	// SDK payload.
	// +optional
	IncludeRedirectExperiments *bool `json:"includeRedirectExperiments,omitempty"`

	// IncludeRuleIds includes rule ids in the SDK payload.
	// +optional
	IncludeRuleIds *bool `json:"includeRuleIds,omitempty"`

	// IncludeProjectIdInMetadata includes the project id in feature
	// metadata.
	// +optional
	IncludeProjectIdInMetadata *bool `json:"includeProjectIdInMetadata,omitempty"`

	// IncludeCustomFieldsInMetadata includes custom fields in feature
	// metadata.
	// +optional
	IncludeCustomFieldsInMetadata *bool `json:"includeCustomFieldsInMetadata,omitempty"`

	// AllowedCustomFieldsInMetadata limits which custom fields are included
	// when IncludeCustomFieldsInMetadata is set.
	// +optional
	AllowedCustomFieldsInMetadata []string `json:"allowedCustomFieldsInMetadata,omitempty"`

	// IncludeTagsInMetadata includes tags in feature metadata.
	// +optional
	IncludeTagsInMetadata *bool `json:"includeTagsInMetadata,omitempty"`

	// IncludeExperimentScheduleInMetadata includes experiment scheduling
	// info in feature metadata.
	// +optional
	IncludeExperimentScheduleInMetadata *bool `json:"includeExperimentScheduleInMetadata,omitempty"`

	// ProxyEnabled enables the GrowthBook proxy for this connection.
	// +optional
	ProxyEnabled *bool `json:"proxyEnabled,omitempty"`

	// ProxyHost is the proxy host to use when ProxyEnabled is true.
	// +optional
	ProxyHost *string `json:"proxyHost,omitempty"`

	// HashSecureAttributes hashes secure attributes before they leave the
	// server.
	// +optional
	HashSecureAttributes *bool `json:"hashSecureAttributes,omitempty"`

	// RemoteEvalEnabled enables remote evaluation for this connection.
	// +optional
	RemoteEvalEnabled *bool `json:"remoteEvalEnabled,omitempty"`

	// SavedGroupReferencesEnabled enables saved group references in the SDK
	// payload.
	// +optional
	SavedGroupReferencesEnabled *bool `json:"savedGroupReferencesEnabled,omitempty"`

	// IncludeReferencedPrerequisites carries prerequisite feature flags into
	// the payload even when they target other projects.
	// +optional
	IncludeReferencedPrerequisites *bool `json:"includeReferencedPrerequisites,omitempty"`
}

// SDKConnectionObservation are the observable, non-secret fields of a
// GrowthBook SDK Connection. The client key and proxy signing key are
// secrets: they are only ever surfaced through connection details, never
// written to status.
type SDKConnectionObservation struct {
	// ID is the GrowthBook-assigned connection id ("sdk_...").
	ID string `json:"id,omitempty"`

	// Organization is the GrowthBook organization id that owns the
	// connection.
	Organization string `json:"organization,omitempty"`

	// Languages is the set of SDK languages GrowthBook reports for this
	// connection.
	Languages []string `json:"languages,omitempty"`

	// Project is the first project GrowthBook associates with this
	// connection, kept for backwards compatibility with older API
	// responses. Prefer Projects on the spec.
	Project string `json:"project,omitempty"`

	// DateCreated is the RFC 3339 creation timestamp reported by GrowthBook.
	DateCreated string `json:"dateCreated,omitempty"`

	// DateUpdated is the RFC 3339 last-update timestamp reported by
	// GrowthBook.
	DateUpdated string `json:"dateUpdated,omitempty"`

	// Connected reports whether GrowthBook has seen this connection make an
	// SDK request at least once.
	// +optional
	Connected *bool `json:"connected,omitempty"`

	// SSEEnabled reports whether server-sent events are enabled for this
	// connection.
	// +optional
	SSEEnabled *bool `json:"sseEnabled,omitempty"`
}

// A SDKConnectionSpec defines the desired state of a SDKConnection.
type SDKConnectionSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              SDKConnectionParameters `json:"forProvider"`

	// InitProvider holds the same fields as forProvider, which are only
	// applied at resource creation. Crossplane ignores later drift in these
	// fields against the external resource.
	// +optional
	InitProvider SDKConnectionInitParameters `json:"initProvider,omitempty"`
}

// A SDKConnectionStatus represents the observed state of a SDKConnection.
type SDKConnectionStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 SDKConnectionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A SDKConnection is a GrowthBook SDK Connection: a client key an
// application uses to fetch feature flag and experiment definitions for one
// environment (and optionally a subset of projects). Applying this resource
// with spec.writeConnectionSecretToRef set writes the resulting client "key"
// (and, when the proxy is enabled, "proxySigningKey" and "proxyHost") into
// the referenced Secret.
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
