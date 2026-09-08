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
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/core/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

// fakeClient implements ProjectClient with pluggable behaviour.
type fakeClient struct {
	get    func(ctx context.Context, id string) (*growthbook.Project, error)
	create func(ctx context.Context, req growthbook.ProjectRequest) (*growthbook.Project, error)
	update func(ctx context.Context, id string, req growthbook.ProjectRequest) (*growthbook.Project, error)
	delete func(ctx context.Context, id string) error
}

func (f *fakeClient) GetProject(ctx context.Context, id string) (*growthbook.Project, error) {
	return f.get(ctx, id)
}

func (f *fakeClient) CreateProject(ctx context.Context, req growthbook.ProjectRequest) (*growthbook.Project, error) {
	return f.create(ctx, req)
}

func (f *fakeClient) UpdateProject(ctx context.Context, id string, req growthbook.ProjectRequest) (*growthbook.Project, error) {
	return f.update(ctx, id, req)
}

func (f *fakeClient) DeleteProject(ctx context.Context, id string) error {
	return f.delete(ctx, id)
}

var (
	errBoom     = errors.New("boom")
	errNotFound = &growthbook.APIError{StatusCode: http.StatusNotFound, Message: "Could not find project"}
)

const projectName = "web"

func ptr[T any](v T) *T { return &v }

func project(name string, mods ...func(*v1alpha1.Project)) *v1alpha1.Project {
	cr := &v1alpha1.Project{}
	cr.SetName(name)
	cr.Spec.ForProvider.Name = name
	for _, m := range mods {
		m(cr)
	}
	return cr
}

func withExternalName(id string) func(*v1alpha1.Project) {
	return func(cr *v1alpha1.Project) { meta.SetExternalName(cr, id) }
}

func withPublicID(id string) func(*v1alpha1.Project) {
	return func(cr *v1alpha1.Project) { cr.Spec.ForProvider.PublicID = ptr(id) }
}

func withSettings(engine, confidence string) func(*v1alpha1.Project) {
	return func(cr *v1alpha1.Project) {
		cr.Spec.ForProvider.Settings = &v1alpha1.ProjectSettings{
			StatsEngine:     ptr(engine),
			ConfidenceLevel: ptr(confidence),
		}
	}
}

func TestObserve(t *testing.T) {
	remote := &growthbook.Project{
		ID: "prj_1", Name: projectName, PublicID: projectName, DateCreated: "2026-09-08T00:00:00Z",
		Settings: &growthbook.ProjectSettings{StatsEngine: ptr("bayesian"), ConfidenceLevel: ptr(0.95)},
	}

	type want struct {
		o        managed.ExternalObservation
		publicID *string
		err      error
	}

	cases := map[string]struct {
		reason string
		client ProjectClient
		cr     *v1alpha1.Project
		want   want
	}{
		"NoExternalName": {
			reason: "A resource without an external name has never been created.",
			client: &fakeClient{},
			cr:     project(projectName),
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"NotFound": {
			reason: "A 404 from the API means the project is gone.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) { return nil, errNotFound }},
			cr:     project(projectName, withExternalName("prj_1")),
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) { return nil, errBoom }},
			cr:     project(projectName, withExternalName("prj_1")),
			want:   want{err: errors.Wrap(errBoom, errGetProject)},
		},
		"UpToDateAndLateInit": {
			reason: "Matching name with unset publicId is up to date and late-initializes publicId.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) { return remote, nil }},
			cr:     project(projectName, withExternalName("prj_1")),
			want: want{
				o:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
				publicID: ptr(projectName),
			},
		},
		"NameDrift": {
			reason: "A different name needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) { return remote, nil }},
			cr: project(projectName, withExternalName("prj_1"), withPublicID(projectName), func(cr *v1alpha1.Project) {
				cr.Spec.ForProvider.Name = "web-renamed"
			}),
			want: want{
				o:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false},
				publicID: ptr(projectName),
			},
		},
		"SettingsMatch": {
			reason: "Decimal settings compare numerically against the API floats.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) { return remote, nil }},
			cr:     project(projectName, withExternalName("prj_1"), withPublicID(projectName), withSettings("bayesian", "0.950")),
			want: want{
				o:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
				publicID: ptr(projectName),
			},
		},
		"SettingsDrift": {
			reason: "A different confidence level needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Project, error) { return remote, nil }},
			cr:     project(projectName, withExternalName("prj_1"), withPublicID(projectName), withSettings("bayesian", "0.9")),
			want: want{
				o:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false},
				publicID: ptr(projectName),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.client}
			got, err := e.Observe(context.Background(), tc.cr)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.publicID, tc.cr.Spec.ForProvider.PublicID); diff != "" {
				t.Errorf("\n%s\ne.Observe(...) publicId: -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	var gotReq growthbook.ProjectRequest
	client := &fakeClient{create: func(_ context.Context, req growthbook.ProjectRequest) (*growthbook.Project, error) {
		gotReq = req
		return &growthbook.Project{ID: "prj_new", Name: req.Name, PublicID: projectName}, nil
	}}
	cr := project(projectName, withSettings("frequentist", "0.9"))

	e := external{client: client}
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("e.Create(...): unexpected error %v", err)
	}

	if got := meta.GetExternalName(cr); got != "prj_new" {
		t.Errorf("external name = %q, want prj_new", got)
	}
	if cr.Status.AtProvider.ID != "prj_new" || cr.Status.AtProvider.PublicID != projectName {
		t.Errorf("atProvider = %+v, want id prj_new and publicId web", cr.Status.AtProvider)
	}
	wantReq := growthbook.ProjectRequest{
		Name:     projectName,
		Settings: &growthbook.ProjectSettings{StatsEngine: ptr("frequentist"), ConfidenceLevel: ptr(0.9)},
	}
	if diff := cmp.Diff(wantReq, gotReq); diff != "" {
		t.Errorf("request body: -want, +got:\n%s", diff)
	}
}

func TestCreateInvalidDecimal(t *testing.T) {
	client := &fakeClient{create: func(_ context.Context, _ growthbook.ProjectRequest) (*growthbook.Project, error) {
		t.Fatal("CreateProject must not be called for an invalid spec")
		return nil, nil
	}}
	cr := project(projectName, withSettings("bayesian", "not-a-number"))

	e := external{client: client}
	if _, err := e.Create(context.Background(), cr); err == nil {
		t.Fatal("e.Create(...): expected error for invalid confidenceLevel")
	}
}

func TestUpdate(t *testing.T) {
	var gotID string
	client := &fakeClient{update: func(_ context.Context, id string, req growthbook.ProjectRequest) (*growthbook.Project, error) {
		gotID = id
		return &growthbook.Project{ID: id, Name: req.Name, DateUpdated: "2026-09-08T01:00:00Z"}, nil
	}}
	cr := project(projectName, withExternalName("prj_1"))

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotID != "prj_1" {
		t.Errorf("updated id = %q, want prj_1", gotID)
	}
	if cr.Status.AtProvider.DateUpdated != "2026-09-08T01:00:00Z" {
		t.Errorf("atProvider.dateUpdated not refreshed: %+v", cr.Status.AtProvider)
	}
}

func TestDelete(t *testing.T) {
	cases := map[string]struct {
		reason string
		client ProjectClient
		cr     *v1alpha1.Project
		err    error
	}{
		"NoExternalName": {
			reason: "Nothing was ever created, so there is nothing to delete.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { t.Fatal("must not call API"); return nil }},
			cr:     project(projectName),
		},
		"AlreadyGone": {
			reason: "A 404 on delete is treated as success.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errNotFound }},
			cr:     project(projectName, withExternalName("prj_1")),
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errBoom }},
			cr:     project(projectName, withExternalName("prj_1")),
			err:    errors.Wrap(errBoom, errDeleteProject),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.client}
			_, err := e.Delete(context.Background(), tc.cr)
			if diff := cmp.Diff(tc.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
