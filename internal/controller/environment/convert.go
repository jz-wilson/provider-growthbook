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

package environment

import (
	"sort"

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/core/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

// createRequest builds the POST body, including the create-only id and
// parent. Fields unset in forProvider fall back to initProvider; forProvider
// wins whenever both set the same field. initProvider fields are only ever
// consulted here, at creation time.
func createRequest(id string, p v1alpha1.EnvironmentParameters, ip v1alpha1.EnvironmentInitParameters) growthbook.EnvironmentRequest {
	merged := mergeInitProvider(p, ip)
	req := updateRequest(merged)
	req.ID = id
	req.Parent = merged.Parent
	return req
}

// mergeInitProvider fills any forProvider field left unset from the
// matching initProvider field. forProvider always wins when both are set.
func mergeInitProvider(p v1alpha1.EnvironmentParameters, ip v1alpha1.EnvironmentInitParameters) v1alpha1.EnvironmentParameters {
	if p.Description == nil {
		p.Description = ip.Description
	}
	if p.ToggleOnList == nil {
		p.ToggleOnList = ip.ToggleOnList
	}
	if p.DefaultState == nil {
		p.DefaultState = ip.DefaultState
	}
	if p.Projects == nil {
		p.Projects = ip.Projects
	}
	if p.Parent == nil {
		p.Parent = ip.Parent
	}
	return p
}

// updateRequest builds the PUT body from the mutable fields only.
func updateRequest(p v1alpha1.EnvironmentParameters) growthbook.EnvironmentRequest {
	return growthbook.EnvironmentRequest{
		Description:  p.Description,
		ToggleOnList: p.ToggleOnList,
		DefaultState: p.DefaultState,
		Projects:     p.Projects,
	}
}

// observation maps the API object onto status.atProvider.
func observation(e *growthbook.Environment) v1alpha1.EnvironmentObservation {
	return v1alpha1.EnvironmentObservation{ID: e.ID, Parent: e.Parent}
}

// lateInitialize fills optional fields the user left unset from the API so
// the spec reflects what GrowthBook actually holds. It reports whether the
// spec changed. A field the user set in spec.initProvider is never
// late-initialized: copying it into forProvider would make it enforced and
// turn later API-side changes into drift, breaking the create-only contract.
func lateInitialize(p *v1alpha1.EnvironmentParameters, ip v1alpha1.EnvironmentInitParameters, ext *growthbook.Environment) bool {
	changed := lateInitPtr(&p.Description, ip.Description, ext.Description)
	changed = lateInitPtr(&p.ToggleOnList, ip.ToggleOnList, ext.ToggleOnList) || changed
	changed = lateInitPtr(&p.DefaultState, ip.DefaultState, ext.DefaultState) || changed
	if ext.Parent != "" {
		changed = lateInitPtr(&p.Parent, ip.Parent, ext.Parent) || changed
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

// isUpToDate compares the mutable fields. Unset (nil) projects means the
// user does not manage the list; a set list compares as a set.
func isUpToDate(p v1alpha1.EnvironmentParameters, ext *growthbook.Environment) bool {
	switch {
	case p.Description != nil && *p.Description != ext.Description:
		return false
	case p.ToggleOnList != nil && *p.ToggleOnList != ext.ToggleOnList:
		return false
	case p.DefaultState != nil && *p.DefaultState != ext.DefaultState:
		return false
	case p.Projects == nil:
		return true
	}
	return sameSet(p.Projects, ext.Projects)
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
