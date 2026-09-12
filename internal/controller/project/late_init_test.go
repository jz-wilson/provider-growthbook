/*
Copyright 2026 The provider-growthbook Authors.

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
	"context"
	"testing"

	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

// TestInitProviderSurvivesLateInitialization walks the full reconcile
// sequence: after Create, the first Observe must not late-initialize a
// field the user set only in spec.initProvider into spec.forProvider.
// Otherwise the field becomes enforced and the next API-side change to it
// is reported as drift, which defeats initProvider's create-only contract.
func TestInitProviderSurvivesLateInitialization(t *testing.T) {
	cr := project(projectName, withExternalName(projID), withInitPublicID("init-slug"))

	apiPublicID := "init-slug"
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) {
		return &growthbook.Project{ID: projID, Name: projectName, PublicID: apiPublicID}, nil
	}}
	e := external{client: client}

	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatalf("first e.Observe(...): unexpected error %v", err)
	}
	if cr.Spec.ForProvider.PublicID != nil {
		t.Fatalf("late-initialization copied initProvider-only publicId into forProvider: %q", *cr.Spec.ForProvider.PublicID)
	}

	apiPublicID = "renamed-in-growthbook"
	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("second e.Observe(...): unexpected error %v", err)
	}
	if !got.ResourceUpToDate {
		t.Errorf("an API-side change to an initProvider-only field must not be reported as drift")
	}
}
