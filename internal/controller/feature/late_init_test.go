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

package feature

import (
	"context"
	"testing"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// TestInitProviderSurvivesLateInitialization walks the full reconcile
// sequence: after Create, the first Observe must not late-initialize
// fields the user set only in spec.initProvider (description, owner,
// project) into spec.forProvider. Otherwise they become enforced and the
// next API-side change is reported as drift, which defeats initProvider's
// create-only contract.
func TestInitProviderSurvivesLateInitialization(t *testing.T) {
	cr := newFeature(withInitDescription("init desc"), func(f *v1alpha1.Feature) {
		f.Spec.InitProvider.Owner = ptr("init-owner")
		f.Spec.InitProvider.Project = ptr("prj_init")
	})

	api := struct{ description, owner, project string }{"init desc", "init-owner", "prj_init"}
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) {
		r := remote()
		r.Description = api.description
		r.Owner = api.owner
		r.Project = api.project
		return r, nil
	}}
	e := external{client: client}

	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatalf("first e.Observe(...): unexpected error %v", err)
	}
	fp := cr.Spec.ForProvider
	if fp.Description != nil || fp.Owner != nil || fp.Project != nil {
		t.Fatalf("late-initialization copied initProvider-only fields into forProvider: description=%v owner=%v project=%v",
			fp.Description, fp.Owner, fp.Project)
	}

	api.description, api.owner, api.project = "edited in the UI", "someone-else", "prj_moved"
	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("second e.Observe(...): unexpected error %v", err)
	}
	if !got.ResourceUpToDate {
		t.Errorf("API-side changes to initProvider-only fields must not be reported as drift")
	}
}
