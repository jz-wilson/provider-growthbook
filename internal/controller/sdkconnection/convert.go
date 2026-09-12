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

package sdkconnection

import (
	"sort"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/sdk/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// createRequest converts the desired spec into the create request body,
// filling any forProvider field left unset from the matching initProvider
// field. forProvider always wins when both are set. initProvider is only
// ever consulted here, at creation time.
func createRequest(p v1alpha1.SDKConnectionParameters, ip v1alpha1.SDKConnectionInitParameters) growthbook.SDKConnectionRequest {
	return request(mergeInitProvider(p, ip))
}

// mergeInitProvider fills any forProvider field left unset from the
// matching initProvider field. forProvider always wins when both are set.
func mergeInitProvider(p v1alpha1.SDKConnectionParameters, ip v1alpha1.SDKConnectionInitParameters) v1alpha1.SDKConnectionParameters {
	mergeIdentityFields(&p, ip)
	mergeScopeFields(&p, ip)
	mergeScalarFields(&p, ip)
	mergeBoolFields(&p, ip)
	return p
}

// mergeIdentityFields merges the always-required string fields.
func mergeIdentityFields(p *v1alpha1.SDKConnectionParameters, ip v1alpha1.SDKConnectionInitParameters) {
	if p.Name == "" && ip.Name != nil {
		p.Name = *ip.Name
	}
	if p.Language == "" && ip.Language != nil {
		p.Language = *ip.Language
	}
	if p.Environment == "" && ip.Environment != nil {
		p.Environment = *ip.Environment
	}
}

// mergeScopeFields merges the optional set-valued fields.
func mergeScopeFields(p *v1alpha1.SDKConnectionParameters, ip v1alpha1.SDKConnectionInitParameters) {
	if p.Projects == nil {
		p.Projects = ip.Projects
	}
	if p.AllowedCustomFieldsInMetadata == nil {
		p.AllowedCustomFieldsInMetadata = ip.AllowedCustomFieldsInMetadata
	}
}

// mergeScalarFields merges the optional non-boolean scalar fields.
func mergeScalarFields(p *v1alpha1.SDKConnectionParameters, ip v1alpha1.SDKConnectionInitParameters) {
	if p.SDKVersion == nil {
		p.SDKVersion = ip.SDKVersion
	}
	if p.ProxyHost == nil {
		p.ProxyHost = ip.ProxyHost
	}
}

// mergeBoolFields merges the optional boolean toggles. Table-driven to keep
// cyclomatic complexity low despite the field count.
func mergeBoolFields(p *v1alpha1.SDKConnectionParameters, ip v1alpha1.SDKConnectionInitParameters) {
	pairs := []struct {
		dst **bool
		src *bool
	}{
		{&p.EncryptPayload, ip.EncryptPayload},
		{&p.IncludeVisualExperiments, ip.IncludeVisualExperiments},
		{&p.IncludeDraftExperiments, ip.IncludeDraftExperiments},
		{&p.IncludeDraftExperimentRefs, ip.IncludeDraftExperimentRefs},
		{&p.IncludeExperimentNames, ip.IncludeExperimentNames},
		{&p.IncludeRedirectExperiments, ip.IncludeRedirectExperiments},
		{&p.IncludeRuleIds, ip.IncludeRuleIds},
		{&p.IncludeProjectIdInMetadata, ip.IncludeProjectIdInMetadata},
		{&p.IncludeCustomFieldsInMetadata, ip.IncludeCustomFieldsInMetadata},
		{&p.IncludeTagsInMetadata, ip.IncludeTagsInMetadata},
		{&p.IncludeExperimentScheduleInMetadata, ip.IncludeExperimentScheduleInMetadata},
		{&p.ProxyEnabled, ip.ProxyEnabled},
		{&p.HashSecureAttributes, ip.HashSecureAttributes},
		{&p.RemoteEvalEnabled, ip.RemoteEvalEnabled},
		{&p.SavedGroupReferencesEnabled, ip.SavedGroupReferencesEnabled},
		{&p.IncludeReferencedPrerequisites, ip.IncludeReferencedPrerequisites},
	}
	for _, pr := range pairs {
		if *pr.dst == nil {
			*pr.dst = pr.src
		}
	}
}

// request converts the desired spec into the API request body.
func request(p v1alpha1.SDKConnectionParameters) growthbook.SDKConnectionRequest {
	return growthbook.SDKConnectionRequest{
		Name:                                p.Name,
		Language:                            p.Language,
		Environment:                         p.Environment,
		SDKVersion:                          p.SDKVersion,
		Projects:                            p.Projects,
		EncryptPayload:                      p.EncryptPayload,
		IncludeVisualExperiments:            p.IncludeVisualExperiments,
		IncludeDraftExperiments:             p.IncludeDraftExperiments,
		IncludeDraftExperimentRefs:          p.IncludeDraftExperimentRefs,
		IncludeExperimentNames:              p.IncludeExperimentNames,
		IncludeRedirectExperiments:          p.IncludeRedirectExperiments,
		IncludeRuleIds:                      p.IncludeRuleIds,
		IncludeProjectIdInMetadata:          p.IncludeProjectIdInMetadata,
		IncludeCustomFieldsInMetadata:       p.IncludeCustomFieldsInMetadata,
		AllowedCustomFieldsInMetadata:       p.AllowedCustomFieldsInMetadata,
		IncludeTagsInMetadata:               p.IncludeTagsInMetadata,
		IncludeExperimentScheduleInMetadata: p.IncludeExperimentScheduleInMetadata,
		ProxyEnabled:                        p.ProxyEnabled,
		ProxyHost:                           p.ProxyHost,
		HashSecureAttributes:                p.HashSecureAttributes,
		RemoteEvalEnabled:                   p.RemoteEvalEnabled,
		SavedGroupReferencesEnabled:         p.SavedGroupReferencesEnabled,
		IncludeReferencedPrerequisites:      p.IncludeReferencedPrerequisites,
	}
}

// observation maps the API object onto status.atProvider.
func observation(sc *growthbook.SDKConnection) v1alpha1.SDKConnectionObservation {
	return v1alpha1.SDKConnectionObservation{
		ID:           sc.ID,
		Organization: sc.Organization,
		Languages:    sc.Languages,
		Project:      sc.Project,
		DateCreated:  sc.DateCreated,
		DateUpdated:  sc.DateUpdated,
		Connected:    sc.Connected,
		SSEEnabled:   sc.SSEEnabled,
	}
}

// isUpToDate compares only the fields the user set. Unset optional fields
// are left to GrowthBook. Split into several small helpers to keep each
// one's cyclomatic complexity low.
func isUpToDate(p v1alpha1.SDKConnectionParameters, sc *growthbook.SDKConnection) bool {
	return identityFieldsUpToDate(p, sc) &&
		scopeFieldsUpToDate(p, sc) &&
		scalarFieldsUpToDate(p, sc) &&
		boolFieldsUpToDate(p, sc)
}

// identityFieldsUpToDate compares the always-required fields.
func identityFieldsUpToDate(p v1alpha1.SDKConnectionParameters, sc *growthbook.SDKConnection) bool {
	return p.Name == sc.Name && p.Environment == sc.Environment && languageMatches(p.Language, sc.Languages)
}

// scopeFieldsUpToDate compares the optional set-valued fields.
func scopeFieldsUpToDate(p v1alpha1.SDKConnectionParameters, sc *growthbook.SDKConnection) bool {
	if p.Projects != nil && !stringSlicesEqual(normalize(p.Projects), normalize(sc.Projects)) {
		return false
	}
	if p.AllowedCustomFieldsInMetadata != nil && !stringSlicesEqual(normalize(p.AllowedCustomFieldsInMetadata), normalize(sc.AllowedCustomFieldsInMetadata)) {
		return false
	}
	return true
}

// scalarFieldsUpToDate compares the optional non-boolean scalar fields.
func scalarFieldsUpToDate(p v1alpha1.SDKConnectionParameters, sc *growthbook.SDKConnection) bool {
	if p.SDKVersion != nil && *p.SDKVersion != sc.SDKVersion {
		return false
	}
	if p.EncryptPayload != nil && *p.EncryptPayload != sc.EncryptPayload {
		return false
	}
	if p.ProxyEnabled != nil && *p.ProxyEnabled != sc.ProxyEnabled {
		return false
	}
	if p.ProxyHost != nil && *p.ProxyHost != sc.ProxyHost {
		return false
	}
	return true
}

// boolFieldsUpToDate compares the optional boolean toggles. Split out from
// isUpToDate to keep cyclomatic complexity low.
func boolFieldsUpToDate(p v1alpha1.SDKConnectionParameters, sc *growthbook.SDKConnection) bool {
	checks := []struct {
		desired *bool
		actual  *bool
	}{
		{p.IncludeVisualExperiments, sc.IncludeVisualExperiments},
		{p.IncludeDraftExperiments, sc.IncludeDraftExperiments},
		{p.IncludeDraftExperimentRefs, sc.IncludeDraftExperimentRefs},
		{p.IncludeExperimentNames, sc.IncludeExperimentNames},
		{p.IncludeRedirectExperiments, sc.IncludeRedirectExperiments},
		{p.IncludeRuleIds, sc.IncludeRuleIds},
		{p.IncludeProjectIdInMetadata, sc.IncludeProjectIdInMetadata},
		{p.IncludeCustomFieldsInMetadata, sc.IncludeCustomFieldsInMetadata},
		{p.IncludeTagsInMetadata, sc.IncludeTagsInMetadata},
		{p.IncludeExperimentScheduleInMetadata, sc.IncludeExperimentScheduleInMetadata},
		{p.HashSecureAttributes, sc.HashSecureAttributes},
		{p.RemoteEvalEnabled, sc.RemoteEvalEnabled},
		{p.SavedGroupReferencesEnabled, sc.SavedGroupReferencesEnabled},
		{p.IncludeReferencedPrerequisites, sc.IncludeReferencedPrerequisites},
	}
	for _, c := range checks {
		if c.desired != nil && (c.actual == nil || *c.desired != *c.actual) {
			return false
		}
	}
	return true
}

// languageMatches reports whether the desired single language is present in
// the API's reported language set. GrowthBook's response reports the
// connection's languages as a slice even though create/update take one.
func languageMatches(desired string, actual []string) bool {
	for _, l := range actual {
		if l == desired {
			return true
		}
	}
	return false
}

func normalize(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
