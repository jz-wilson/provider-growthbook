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

// fakeClient implements EnvironmentClient with pluggable behaviour.
type fakeClient struct {
	get    func(ctx context.Context, id string) (*growthbook.Environment, error)
	create func(ctx context.Context, req growthbook.EnvironmentRequest) (*growthbook.Environment, error)
	update func(ctx context.Context, id string, req growthbook.EnvironmentRequest) (*growthbook.Environment, error)
	delete func(ctx context.Context, id string) error
}

func (f *fakeClient) GetEnvironment(ctx context.Context, id string) (*growthbook.Environment, error) {
	return f.get(ctx, id)
}

func (f *fakeClient) CreateEnvironment(ctx context.Context, req growthbook.EnvironmentRequest) (*growthbook.Environment, error) {
	return f.create(ctx, req)
}

func (f *fakeClient) UpdateEnvironment(ctx context.Context, id string, req growthbook.EnvironmentRequest) (*growthbook.Environment, error) {
	return f.update(ctx, id, req)
}

func (f *fakeClient) DeleteEnvironment(ctx context.Context, id string) error {
	return f.delete(ctx, id)
}

const envID = "staging"

var (
	errBoom     = errors.New("boom")
	errNotFound = &growthbook.APIError{StatusCode: http.StatusNotFound, Message: "environment not found"}
)

func ptr[T any](v T) *T { return &v }

func environment(mods ...func(*v1alpha1.Environment)) *v1alpha1.Environment {
	cr := &v1alpha1.Environment{}
	cr.SetName(envID)
	meta.SetExternalName(cr, envID)
	for _, m := range mods {
		m(cr)
	}
	return cr
}

func withDescription(d string) func(*v1alpha1.Environment) {
	return func(cr *v1alpha1.Environment) { cr.Spec.ForProvider.Description = ptr(d) }
}

func withProjects(p ...string) func(*v1alpha1.Environment) {
	return func(cr *v1alpha1.Environment) { cr.Spec.ForProvider.Projects = p }
}

func withDefaultState(b bool) func(*v1alpha1.Environment) {
	return func(cr *v1alpha1.Environment) { cr.Spec.ForProvider.DefaultState = ptr(b) }
}

func remote() *growthbook.Environment {
	return &growthbook.Environment{ID: envID, Description: "Stage", ToggleOnList: true, DefaultState: false, Projects: []string{"prj_1"}}
}

func TestObserve(t *testing.T) {
	type want struct {
		o           managed.ExternalObservation
		description *string
		err         error
	}

	cases := map[string]struct {
		reason string
		client EnvironmentClient
		cr     *v1alpha1.Environment
		want   want
	}{
		"NotFound": {
			reason: "An environment missing from the list has to be created.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) { return nil, errNotFound }},
			cr:     environment(),
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) { return nil, errBoom }},
			cr:     environment(),
			want:   want{err: errors.Wrap(errBoom, errGetEnvironment)},
		},
		"UpToDateLateInit": {
			reason: "Unset optional fields are late-initialized from the API and count as up to date.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) { return remote(), nil }},
			cr:     environment(),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
				description: ptr("Stage"),
			},
		},
		"DescriptionDrift": {
			reason: "A changed description needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) { return remote(), nil }},
			cr:     environment(withDescription("Staging env")),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: true},
				description: ptr("Staging env"),
			},
		},
		"ProjectsOrderInsensitive": {
			reason: "Projects compare as a set, so order does not trigger an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) {
				r := remote()
				r.Projects = []string{"prj_2", "prj_1"}
				return r, nil
			}},
			cr: environment(withDescription("Stage"), withProjects("prj_1", "prj_2")),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
				description: ptr("Stage"),
			},
		},
		"DefaultStateDrift": {
			reason: "An explicit defaultState that differs needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) { return remote(), nil }},
			cr:     environment(withDescription("Stage"), withDefaultState(true)),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: true},
				description: ptr("Stage"),
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
			if diff := cmp.Diff(tc.want.description, tc.cr.Spec.ForProvider.Description); diff != "" {
				t.Errorf("\n%s\ne.Observe(...) description: -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	var gotReq growthbook.EnvironmentRequest
	client := &fakeClient{create: func(_ context.Context, req growthbook.EnvironmentRequest) (*growthbook.Environment, error) {
		gotReq = req
		return &growthbook.Environment{ID: req.ID, Projects: []string{}}, nil
	}}
	cr := environment(withDescription("Stage"), withProjects("prj_1"), func(cr *v1alpha1.Environment) {
		cr.Spec.ForProvider.Parent = ptr("production")
	})

	e := external{client: client}
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("e.Create(...): unexpected error %v", err)
	}
	wantReq := growthbook.EnvironmentRequest{
		ID: envID, Description: ptr("Stage"), Projects: []string{"prj_1"}, Parent: ptr("production"),
	}
	if diff := cmp.Diff(wantReq, gotReq); diff != "" {
		t.Errorf("request body: -want, +got:\n%s", diff)
	}
	if cr.Status.AtProvider.ID != envID {
		t.Errorf("atProvider.id = %q, want %q", cr.Status.AtProvider.ID, envID)
	}
}

func TestUpdate(t *testing.T) {
	var gotID string
	var gotReq growthbook.EnvironmentRequest
	client := &fakeClient{update: func(_ context.Context, id string, req growthbook.EnvironmentRequest) (*growthbook.Environment, error) {
		gotID, gotReq = id, req
		return remote(), nil
	}}
	cr := environment(withDescription("Stage"), withDefaultState(true), func(cr *v1alpha1.Environment) {
		cr.Spec.ForProvider.Parent = ptr("production")
	})

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotID != envID {
		t.Errorf("updated id = %q, want %q", gotID, envID)
	}
	if gotReq.ID != "" || gotReq.Parent != nil {
		t.Errorf("Update must not send id or parent (create-only): %+v", gotReq)
	}
	if gotReq.DefaultState == nil || !*gotReq.DefaultState {
		t.Errorf("Update must send defaultState=true: %+v", gotReq)
	}
}

func TestDelete(t *testing.T) {
	cases := map[string]struct {
		reason string
		client EnvironmentClient
		err    error
	}{
		"AlreadyGone": {
			reason: "A 404 on delete is treated as success.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errNotFound }},
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errBoom }},
			err:    errors.Wrap(errBoom, errDeleteEnvironment),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.client}
			_, err := e.Delete(context.Background(), environment())
			if diff := cmp.Diff(tc.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
