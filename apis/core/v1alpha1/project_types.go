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

// ProjectSettings override the organization's statistics settings for one
// project. Decimal values are strings because CRD schemas forbid floating
// point numbers; the controller converts them.
type ProjectSettings struct {
	// StatsEngine selects the statistics engine, for example "bayesian" or
	// "frequentist".
	// +optional
	StatsEngine *string `json:"statsEngine,omitempty"`

	// ConfidenceLevel is the Bayesian chance-to-win threshold as a decimal
	// string, for example "0.95".
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`
	ConfidenceLevel *string `json:"confidenceLevel,omitempty"`

	// PValueThreshold is the frequentist p-value threshold as a decimal
	// string, for example "0.05".
	// +optional
	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`
	PValueThreshold *string `json:"pValueThreshold,omitempty"`
}

// ProjectParameters are the configurable fields of a GrowthBook Project.
type ProjectParameters struct {
	// Name is the human-readable project name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is free text shown in the GrowthBook UI.
	// +optional
	// +kubebuilder:validation:MaxLength=10000
	Description *string `json:"description,omitempty"`

	// PublicID is the URL-safe slug (lowercase letters, numbers, dashes)
	// used in SDK payload metadata. GrowthBook derives one from Name when
	// unset; the derived value is then late-initialized here.
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9-]+$`
	PublicID *string `json:"publicId,omitempty"`

	// RestrictAccess limits the project to members with an explicit role on
	// it. Requires a GrowthBook Pro or Enterprise plan.
	// +optional
	RestrictAccess *bool `json:"restrictAccess,omitempty"`

	// Settings override organization statistics settings for this project.
	// +optional
	Settings *ProjectSettings `json:"settings,omitempty"`
}

// ProjectObservation are the observable fields of a GrowthBook Project.
type ProjectObservation struct {
	// ID is the GrowthBook-assigned project id ("prj_...").
	ID string `json:"id,omitempty"`

	// PublicID is the slug GrowthBook stores for the project.
	PublicID string `json:"publicId,omitempty"`

	// DateCreated is the RFC 3339 creation timestamp reported by GrowthBook.
	DateCreated string `json:"dateCreated,omitempty"`

	// DateUpdated is the RFC 3339 last-update timestamp reported by GrowthBook.
	DateUpdated string `json:"dateUpdated,omitempty"`
}

// A ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectParameters `json:"forProvider"`
}

// A ProjectStatus represents the observed state of a Project.
type ProjectStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 ProjectObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Project is a GrowthBook project: the top-level container for features,
// experiments, and SDK connections.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,growthbook}
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// Project type metadata.
var (
	ProjectKind             = reflect.TypeOf(Project{}).Name()
	ProjectGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectKind}.String()
	ProjectKindAPIVersion   = ProjectKind + "." + SchemeGroupVersion.String()
	ProjectGroupVersionKind = SchemeGroupVersion.WithKind(ProjectKind)
)

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
