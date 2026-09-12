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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/core/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

func withInitDescription(d string) func(*v1alpha1.Project) {
	return func(cr *v1alpha1.Project) { cr.Spec.InitProvider.Description = ptr(d) }
}

func withInitPublicID(id string) func(*v1alpha1.Project) {
	return func(cr *v1alpha1.Project) { cr.Spec.InitProvider.PublicID = ptr(id) }
}

// TestCreateInitProvider covers the spec.initProvider merge semantics on
// Create: a field set only in initProvider is sent, and forProvider wins
// when both are set.
func TestCreateInitProvider(t *testing.T) {
	cases := map[string]struct {
		reason  string
		cr      *v1alpha1.Project
		wantReq growthbook.ProjectRequest
	}{
		"InitProviderOnlyFieldIsSent": {
			reason: "A field set only in initProvider must be sent on Create.",
			cr:     project(projectName, withInitDescription("init desc"), withInitPublicID("init-web")),
			wantReq: growthbook.ProjectRequest{
				Name: projectName, Description: ptr("init desc"), PublicID: ptr("init-web"),
			},
		},
		"ForProviderOverridesInitProvider": {
			reason: "forProvider wins over initProvider when both set the same field.",
			cr: project(projectName,
				withInitDescription("init desc"),
				func(cr *v1alpha1.Project) { cr.Spec.ForProvider.Description = ptr("forProvider desc") },
			),
			wantReq: growthbook.ProjectRequest{
				Name: projectName, Description: ptr("forProvider desc"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var gotReq growthbook.ProjectRequest
			client := &fakeClient{create: func(_ context.Context, req growthbook.ProjectRequest) (*growthbook.Project, error) {
				gotReq = req
				return &growthbook.Project{ID: projID, Name: req.Name}, nil
			}}
			e := external{client: client}
			if _, err := e.Create(context.Background(), tc.cr); err != nil {
				t.Fatalf("e.Create(...): unexpected error %v", err)
			}
			if diff := cmp.Diff(tc.wantReq, gotReq); diff != "" {
				t.Errorf("\n%s\nrequest body: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestObserveIgnoresInitProviderDrift asserts that a later API-side change
// to a field that was only ever set via initProvider never marks the
// resource out of date, and is not copied into initProvider by
// late-initialization.
func TestObserveIgnoresInitProviderDrift(t *testing.T) {
	cr := project(projectName, withInitPublicID("init-web"), withExternalName(projID))
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) {
		return &growthbook.Project{ID: projID, Name: projectName, PublicID: "changed-out-from-under-us"}, nil
	}}

	e := external{client: client}
	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("e.Observe(...): unexpected error %v", err)
	}
	// publicId is set only in initProvider, so late-initialization must leave
	// forProvider.publicId unset and report nothing late-initialized.
	want := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: false}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("initProvider-only field must never cause drift: -want, +got:\n%s", diff)
	}
	if cr.Spec.ForProvider.PublicID != nil {
		t.Errorf("late-initialization must not copy initProvider-only publicId into forProvider: got %q", *cr.Spec.ForProvider.PublicID)
	}
	if cr.Spec.InitProvider.PublicID == nil || *cr.Spec.InitProvider.PublicID != "init-web" {
		t.Errorf("late-initialization must not overwrite initProvider: got %v", cr.Spec.InitProvider.PublicID)
	}
}

// TestUpdateNeverSendsInitProviderOnlyValues asserts Update never leaks
// initProvider-only field values into the PUT body.
func TestUpdateNeverSendsInitProviderOnlyValues(t *testing.T) {
	var gotReq growthbook.ProjectRequest
	client := &fakeClient{update: func(_ context.Context, _ string, req growthbook.ProjectRequest) (*growthbook.Project, error) {
		gotReq = req
		return &growthbook.Project{ID: projID, Name: projectName}, nil
	}}
	cr := project(projectName, withInitDescription("init desc"), withInitPublicID("init-web"), withExternalName(projID))

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotReq.Description != nil {
		t.Errorf("Update must not send initProvider-only description: %+v", gotReq)
	}
	if gotReq.PublicID != nil {
		t.Errorf("Update must not send initProvider-only publicId: %+v", gotReq)
	}
	if meta.GetExternalName(cr) != projID {
		t.Errorf("unexpected external name mutation: %q", meta.GetExternalName(cr))
	}
}
