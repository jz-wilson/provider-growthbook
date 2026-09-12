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

package project

import (
	"strconv"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/core/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// createRequest converts the desired spec into the create request body,
// filling any forProvider field left unset from the matching initProvider
// field. forProvider always wins when both are set. initProvider is only
// ever consulted here, at creation time.
func createRequest(p v1alpha1.ProjectParameters, ip v1alpha1.ProjectInitParameters) (growthbook.ProjectRequest, error) {
	return request(mergeInitProvider(p, ip))
}

func mergeInitProvider(p v1alpha1.ProjectParameters, ip v1alpha1.ProjectInitParameters) v1alpha1.ProjectParameters {
	if p.Name == "" && ip.Name != nil {
		p.Name = *ip.Name
	}
	if p.Description == nil {
		p.Description = ip.Description
	}
	if p.PublicID == nil {
		p.PublicID = ip.PublicID
	}
	if p.RestrictAccess == nil {
		p.RestrictAccess = ip.RestrictAccess
	}
	if p.Settings == nil {
		p.Settings = ip.Settings
	}
	return p
}

// request converts the desired spec into the API request body.
func request(p v1alpha1.ProjectParameters) (growthbook.ProjectRequest, error) {
	req := growthbook.ProjectRequest{
		Name:           p.Name,
		Description:    p.Description,
		PublicID:       p.PublicID,
		RestrictAccess: p.RestrictAccess,
	}
	if p.Settings == nil {
		return req, nil
	}

	s := &growthbook.ProjectSettings{StatsEngine: p.Settings.StatsEngine}
	var err error
	if s.ConfidenceLevel, err = parseDecimal("confidenceLevel", p.Settings.ConfidenceLevel); err != nil {
		return req, err
	}
	if s.PValueThreshold, err = parseDecimal("pValueThreshold", p.Settings.PValueThreshold); err != nil {
		return req, err
	}
	req.Settings = s
	return req, nil
}

func parseDecimal(field string, s *string) (*float64, error) {
	if s == nil {
		return nil, nil
	}
	f, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "%s is not a decimal", field)
	}
	return &f, nil
}

// observation maps the API object onto status.atProvider.
func observation(p *growthbook.Project) v1alpha1.ProjectObservation {
	return v1alpha1.ProjectObservation{
		ID:          p.ID,
		PublicID:    p.PublicID,
		DateCreated: p.DateCreated,
		DateUpdated: p.DateUpdated,
	}
}

// lateInitialize fills server-generated fields the user left unset. It
// reports whether the spec changed. A field the user set in
// spec.initProvider is never late-initialized: copying it into forProvider
// would make it enforced and turn later API-side changes into drift,
// breaking the create-only contract.
func lateInitialize(p *v1alpha1.ProjectParameters, ip v1alpha1.ProjectInitParameters, ext *growthbook.Project) bool {
	if p.PublicID != nil || ip.PublicID != nil || ext.PublicID == "" {
		return false
	}
	v := ext.PublicID
	p.PublicID = &v
	return true
}

// isUpToDate compares only the fields the user set. Unset optional fields
// are left to GrowthBook.
func isUpToDate(p v1alpha1.ProjectParameters, ext *growthbook.Project) bool {
	return fieldsUpToDate(p, ext) && settingsUpToDate(p.Settings, ext.Settings)
}

func fieldsUpToDate(p v1alpha1.ProjectParameters, ext *growthbook.Project) bool {
	switch {
	case p.Name != ext.Name:
		return false
	case p.Description != nil && *p.Description != ext.Description:
		return false
	case p.PublicID != nil && *p.PublicID != ext.PublicID:
		return false
	case p.RestrictAccess != nil && *p.RestrictAccess != derefBool(ext.RestrictAccess):
		return false
	}
	return true
}

func settingsUpToDate(desired *v1alpha1.ProjectSettings, actual *growthbook.ProjectSettings) bool {
	if desired == nil {
		return true
	}
	if actual == nil {
		return settingsEmpty(desired)
	}
	if desired.StatsEngine != nil && *desired.StatsEngine != derefString(actual.StatsEngine) {
		return false
	}
	return decimalEqual(desired.ConfidenceLevel, actual.ConfidenceLevel) &&
		decimalEqual(desired.PValueThreshold, actual.PValueThreshold)
}

func settingsEmpty(s *v1alpha1.ProjectSettings) bool {
	return s.StatsEngine == nil && s.ConfidenceLevel == nil && s.PValueThreshold == nil
}

// decimalEqual treats a nil desired value as "don't care". An unparseable
// desired value counts as out of date so Update surfaces the error.
func decimalEqual(desired *string, actual *float64) bool {
	if desired == nil {
		return true
	}
	if actual == nil {
		return false
	}
	f, err := strconv.ParseFloat(*desired, 64)
	if err != nil {
		return false
	}
	return f == *actual
}

func derefBool(b *bool) bool {
	return b != nil && *b
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
