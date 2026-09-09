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
// parent.
func createRequest(id string, p v1alpha1.EnvironmentParameters) growthbook.EnvironmentRequest {
	req := updateRequest(p)
	req.ID = id
	req.Parent = p.Parent
	return req
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
// spec changed.
func lateInitialize(p *v1alpha1.EnvironmentParameters, ext *growthbook.Environment) bool {
	changed := false
	if p.Description == nil {
		v := ext.Description
		p.Description = &v
		changed = true
	}
	if p.ToggleOnList == nil {
		v := ext.ToggleOnList
		p.ToggleOnList = &v
		changed = true
	}
	if p.DefaultState == nil {
		v := ext.DefaultState
		p.DefaultState = &v
		changed = true
	}
	if p.Parent == nil && ext.Parent != "" {
		v := ext.Parent
		p.Parent = &v
		changed = true
	}
	return changed
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
