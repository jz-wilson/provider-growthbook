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

package feature

import (
	"sort"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// createRequest builds the POST body, including the create-only id and
// valueType. Fields unset in forProvider fall back to initProvider;
// forProvider wins whenever both set the same field. initProvider fields
// are only ever consulted here, at creation time.
func createRequest(id string, p v1alpha1.FeatureParameters, ip v1alpha1.FeatureInitParameters) (growthbook.FeatureRequest, error) {
	merged := mergeInitProvider(p, ip)
	req, err := updateRequest(merged)
	if err != nil {
		return req, err
	}
	req.ID = id
	req.ValueType = merged.ValueType
	return req, nil
}

// mergeInitProvider fills any forProvider field left unset from the
// matching initProvider field. forProvider always wins when both are set.
// Split into smaller helpers to keep cyclomatic complexity low.
func mergeInitProvider(p v1alpha1.FeatureParameters, ip v1alpha1.FeatureInitParameters) v1alpha1.FeatureParameters {
	mergeValueFields(&p, ip)
	mergeMetadataFields(&p, ip)
	return p
}

// mergeValueFields merges the always-required value fields.
func mergeValueFields(p *v1alpha1.FeatureParameters, ip v1alpha1.FeatureInitParameters) {
	if p.ValueType == "" && ip.ValueType != nil {
		p.ValueType = *ip.ValueType
	}
	if p.DefaultValue == "" && ip.DefaultValue != nil {
		p.DefaultValue = *ip.DefaultValue
	}
}

// mergeMetadataFields merges the optional descriptive fields, environments,
// and rules.
func mergeMetadataFields(p *v1alpha1.FeatureParameters, ip v1alpha1.FeatureInitParameters) {
	if p.Description == nil {
		p.Description = ip.Description
	}
	if p.Project == nil {
		p.Project = ip.Project
	}
	if p.Tags == nil {
		p.Tags = ip.Tags
	}
	if p.Archived == nil {
		p.Archived = ip.Archived
	}
	if p.Owner == nil {
		p.Owner = ip.Owner
	}
	if p.Environments == nil {
		p.Environments = ip.Environments
	}
	if p.Rules == nil {
		p.Rules = ip.Rules
	}
}

// updateRequest builds the update body from the mutable fields only.
func updateRequest(p v1alpha1.FeatureParameters) (growthbook.FeatureRequest, error) {
	rules, err := rulesRequest(p.Rules)
	if err != nil {
		return growthbook.FeatureRequest{}, err
	}
	return growthbook.FeatureRequest{
		DefaultValue: p.DefaultValue,
		Description:  p.Description,
		Project:      p.Project,
		Tags:         p.Tags,
		Archived:     p.Archived,
		Owner:        p.Owner,
		Environments: environmentsRequest(p.Environments),
		Rules:        rules,
	}, nil
}

// environmentsRequest converts the user-set environment map to its request
// shape. A nil map stays nil so an update never touches environments the
// user did not configure.
func environmentsRequest(envs map[string]v1alpha1.FeatureEnvironment) map[string]growthbook.FeatureEnvironmentRequest {
	if envs == nil {
		return nil
	}
	out := make(map[string]growthbook.FeatureEnvironmentRequest, len(envs))
	for k, v := range envs {
		out[k] = growthbook.FeatureEnvironmentRequest{Enabled: v.Enabled}
	}
	return out
}

// observation maps the API object onto status.atProvider.
func observation(f *growthbook.Feature) v1alpha1.FeatureObservation {
	obs := v1alpha1.FeatureObservation{
		ID:          f.ID,
		DateCreated: f.DateCreated,
		DateUpdated: f.DateUpdated,
	}
	if f.Revision != nil {
		obs.Revision = v1alpha1.FeatureRevisionObservation{Version: f.Revision.Version}
	}
	if f.Environments != nil {
		obs.Environments = make(map[string]v1alpha1.FeatureEnvironmentObservation, len(f.Environments))
		for k, v := range f.Environments {
			obs.Environments[k] = v1alpha1.FeatureEnvironmentObservation{Enabled: v.Enabled}
		}
	}
	obs.Rules = rulesObservation(f.Rules)
	return obs
}

// lateInitialize fills optional fields the user left unset from the API so
// the spec reflects what GrowthBook actually holds. It reports whether the
// spec changed. A field the user set in spec.initProvider is never
// late-initialized: copying it into forProvider would make it enforced and
// turn later API-side changes into drift, breaking the create-only contract.
func lateInitialize(p *v1alpha1.FeatureParameters, ip v1alpha1.FeatureInitParameters, ext *growthbook.Feature) bool {
	changed := lateInitPtr(&p.Description, ip.Description, ext.Description)
	if ext.Owner != "" {
		changed = lateInitPtr(&p.Owner, ip.Owner, ext.Owner) || changed
	}
	if ext.Project != "" {
		changed = lateInitPtr(&p.Project, ip.Project, ext.Project) || changed
	}
	return changed
}

// lateInitPtr sets *dst to v when the user left the field unset in both
// forProvider and initProvider. It reports whether it wrote.
func lateInitPtr[T any](dst **T, init *T, v T) bool {
	if *dst != nil || init != nil {
		return false
	}
	*dst = &v
	return true
}

// isUpToDate compares only the fields the user set. Tags compare as a set.
// Environments compare per key the user set; a key the user never
// configured is never inspected, so out-of-band changes to it cannot
// trigger drift.
func isUpToDate(p v1alpha1.FeatureParameters, ext *growthbook.Feature) bool {
	return valueUpToDate(p, ext) && metadataUpToDate(p, ext) &&
		environmentsUpToDate(p.Environments, ext.Environments) && rulesUpToDate(p.Rules, ext.Rules)
}

// valueUpToDate compares the feature's value fields.
func valueUpToDate(p v1alpha1.FeatureParameters, ext *growthbook.Feature) bool {
	return p.ValueType == ext.ValueType && p.DefaultValue == ext.DefaultValue
}

// metadataUpToDate compares the optional descriptive fields the user set.
func metadataUpToDate(p v1alpha1.FeatureParameters, ext *growthbook.Feature) bool {
	switch {
	case p.Description != nil && *p.Description != ext.Description:
		return false
	case p.Project != nil && *p.Project != ext.Project:
		return false
	}
	return ownershipUpToDate(p, ext)
}

// ownershipUpToDate compares archived, owner, and tags.
func ownershipUpToDate(p v1alpha1.FeatureParameters, ext *growthbook.Feature) bool {
	switch {
	case p.Archived != nil && *p.Archived != ext.Archived:
		return false
	case p.Owner != nil && *p.Owner != ext.Owner:
		return false
	case p.Tags != nil && !sameSet(p.Tags, ext.Tags):
		return false
	}
	return true
}

func environmentsUpToDate(want map[string]v1alpha1.FeatureEnvironment, got map[string]growthbook.FeatureEnvironment) bool {
	for k, w := range want {
		if w.Enabled == nil {
			continue
		}
		g, ok := got[k]
		if !ok || g.Enabled != *w.Enabled {
			return false
		}
	}
	return true
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}
