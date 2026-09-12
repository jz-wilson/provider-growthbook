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
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// fakeClient implements FeatureClient with pluggable behaviour.
type fakeClient struct {
	get    func(ctx context.Context, id string) (*growthbook.Feature, error)
	create func(ctx context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error)
	update func(ctx context.Context, id string, req growthbook.FeatureRequest) (*growthbook.Feature, error)
	delete func(ctx context.Context, id string) error
}

func (f *fakeClient) GetFeature(ctx context.Context, id string) (*growthbook.Feature, error) {
	return f.get(ctx, id)
}

func (f *fakeClient) CreateFeature(ctx context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
	return f.create(ctx, req)
}

func (f *fakeClient) UpdateFeature(ctx context.Context, id string, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
	return f.update(ctx, id, req)
}

func (f *fakeClient) DeleteFeature(ctx context.Context, id string) error {
	return f.delete(ctx, id)
}

const featureID = "my-feature"

var (
	errBoom            = errors.New("boom")
	errNotFound        = &growthbook.APIError{StatusCode: http.StatusNotFound, Message: "feature not found"}
	errArchiveRequired = &growthbook.APIError{StatusCode: http.StatusForbidden, Message: "Archive the feature first"}
)

func ptr[T any](v T) *T { return &v }

func newFeature(mods ...func(*v1alpha1.Feature)) *v1alpha1.Feature {
	cr := &v1alpha1.Feature{}
	cr.SetName(featureID)
	meta.SetExternalName(cr, featureID)
	cr.Spec.ForProvider.ValueType = valTypeBoolean
	cr.Spec.ForProvider.DefaultValue = valTrue
	for _, m := range mods {
		m(cr)
	}
	return cr
}

func withDescription(d string) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.ForProvider.Description = ptr(d) }
}

func withTags(tags ...string) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.ForProvider.Tags = tags }
}

func withArchived(b bool) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) { cr.Spec.ForProvider.Archived = ptr(b) }
}

func withEnvironment(name string, enabled bool) func(*v1alpha1.Feature) {
	return func(cr *v1alpha1.Feature) {
		if cr.Spec.ForProvider.Environments == nil {
			cr.Spec.ForProvider.Environments = map[string]v1alpha1.FeatureEnvironment{}
		}
		cr.Spec.ForProvider.Environments[name] = v1alpha1.FeatureEnvironment{Enabled: ptr(enabled)}
	}
}

func remote() *growthbook.Feature {
	return &growthbook.Feature{
		ID:           featureID,
		ValueType:    valTypeBoolean,
		DefaultValue: valTrue,
		Description:  "A feature",
		Owner:        "alice",
		Project:      "prj_1",
		Tags:         []string{"a", "b"},
		Archived:     false,
		Environments: map[string]growthbook.FeatureEnvironment{
			envProduction: {Enabled: true},
			envStaging:    {Enabled: false},
		},
		DateCreated: "2026-01-01T00:00:00Z",
		DateUpdated: "2026-01-02T00:00:00Z",
		Revision:    &growthbook.FeatureRevision{Version: 3},
	}
}

func TestObserve(t *testing.T) {
	type want struct {
		o           managed.ExternalObservation
		description *string
		err         error
	}

	cases := map[string]struct {
		reason string
		client FeatureClient
		cr     *v1alpha1.Feature
		want   want
	}{
		"NotFound": {
			reason: "A feature missing from the API has to be created.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return nil, errNotFound }},
			cr:     newFeature(),
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return nil, errBoom }},
			cr:     newFeature(),
			want:   want{err: errors.Wrap(errBoom, errGetFeature)},
		},
		"UpToDateLateInit": {
			reason: "Unset optional fields are late-initialized from the API and count as up to date.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return remote(), nil }},
			cr:     newFeature(withTags("a", "b"), withEnvironment(envProduction, true), withEnvironment(envStaging, false)),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
				description: ptr("A feature"),
			},
		},
		"DescriptionDrift": {
			reason: "A changed description needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return remote(), nil }},
			cr:     newFeature(withDescription("A different feature")),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: true},
				description: ptr("A different feature"),
			},
		},
		"TagsOrderInsensitive": {
			reason: "Tags compare as a set, so order does not trigger an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) {
				r := remote()
				r.Tags = []string{"b", "a"}
				return r, nil
			}},
			cr: newFeature(withDescription("A feature"), withTags("a", "b")),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
				description: ptr("A feature"),
			},
		},
		"EnvironmentDrift": {
			reason: "An environment enabled flag the user set that differs from the API needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return remote(), nil }},
			cr:     newFeature(withDescription("A feature"), withTags("a", "b"), withEnvironment(envProduction, false)),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: true},
				description: ptr("A feature"),
			},
		},
		"UnmanagedEnvironmentIgnored": {
			reason: "An environment the user never set on the spec is not compared, so it cannot drift.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return remote(), nil }},
			cr:     newFeature(withDescription("A feature"), withTags("a", "b"), withEnvironment(envProduction, true)),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
				description: ptr("A feature"),
			},
		},
		"ArchivedDrift": {
			reason: "An explicit archived flag that differs needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return remote(), nil }},
			cr:     newFeature(withDescription("A feature"), withTags("a", "b"), withArchived(true)),
			want: want{
				o:           managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ResourceLateInitialized: true},
				description: ptr("A feature"),
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

func TestObserveStatus(t *testing.T) {
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Feature, error) { return remote(), nil }}
	cr := newFeature(withDescription("A feature"), withTags("a", "b"))

	e := external{client: client}
	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatalf("e.Observe(...): unexpected error %v", err)
	}
	if cr.Status.AtProvider.ID != featureID {
		t.Errorf("atProvider.id = %q, want %q", cr.Status.AtProvider.ID, featureID)
	}
	if cr.Status.AtProvider.Revision.Version != 3 {
		t.Errorf("atProvider.revision.version = %d, want 3", cr.Status.AtProvider.Revision.Version)
	}
	if cr.Status.AtProvider.DateCreated != "2026-01-01T00:00:00Z" || cr.Status.AtProvider.DateUpdated != "2026-01-02T00:00:00Z" {
		t.Errorf("atProvider dates = %+v", cr.Status.AtProvider)
	}
	if !cr.Status.AtProvider.Environments[envProduction].Enabled || cr.Status.AtProvider.Environments[envStaging].Enabled {
		t.Errorf("atProvider.environments = %+v", cr.Status.AtProvider.Environments)
	}
}

func TestCreate(t *testing.T) {
	var gotReq growthbook.FeatureRequest
	client := &fakeClient{create: func(_ context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
		gotReq = req
		return &growthbook.Feature{ID: req.ID, ValueType: req.ValueType, DefaultValue: req.DefaultValue}, nil
	}}
	cr := newFeature(withDescription("A feature"), withTags("a"), withEnvironment(envProduction, true))

	e := external{client: client}
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatalf("e.Create(...): unexpected error %v", err)
	}
	wantReq := growthbook.FeatureRequest{
		ID: featureID, ValueType: valTypeBoolean, DefaultValue: valTrue, Description: ptr("A feature"), Tags: []string{"a"},
		Environments: map[string]growthbook.FeatureEnvironmentRequest{envProduction: {Enabled: ptr(true)}},
	}
	if diff := cmp.Diff(wantReq, gotReq); diff != "" {
		t.Errorf("request body: -want, +got:\n%s", diff)
	}
	if cr.Status.AtProvider.ID != featureID {
		t.Errorf("atProvider.id = %q, want %q", cr.Status.AtProvider.ID, featureID)
	}
}

func TestUpdate(t *testing.T) {
	var gotID string
	var gotReq growthbook.FeatureRequest
	client := &fakeClient{update: func(_ context.Context, id string, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
		gotID, gotReq = id, req
		return remote(), nil
	}}
	cr := newFeature(withDescription("A feature"), withArchived(true), withEnvironment(envProduction, true))

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotID != featureID {
		t.Errorf("updated id = %q, want %q", gotID, featureID)
	}
	if gotReq.ID != "" || gotReq.ValueType != "" {
		t.Errorf("Update must not send id or valueType (immutable): %+v", gotReq)
	}
	if gotReq.Archived == nil || !*gotReq.Archived {
		t.Errorf("Update must send archived=true: %+v", gotReq)
	}
}

func TestDelete(t *testing.T) {
	cases := map[string]struct {
		reason string
		client FeatureClient
		err    error
	}{
		"AlreadyGone": {
			reason: "A 404 on delete is treated as success.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errNotFound }},
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errBoom }},
			err:    errors.Wrap(errBoom, errDeleteFeature),
		},
		"ArchiveThenDeleteSucceeds": {
			reason: "A 403 asking to archive first is handled by archiving and retrying the delete.",
			client: func() FeatureClient {
				calls := 0
				return &fakeClient{
					delete: func(_ context.Context, _ string) error {
						calls++
						if calls == 1 {
							return errArchiveRequired
						}
						return nil
					},
					update: func(_ context.Context, _ string, req growthbook.FeatureRequest) (*growthbook.Feature, error) {
						if req.Archived == nil || !*req.Archived {
							t.Fatalf("archive update must send archived=true: %+v", req)
						}
						return remote(), nil
					},
				}
			}(),
		},
		"ArchiveFails": {
			reason: "A failed archive attempt is wrapped and returned.",
			client: &fakeClient{
				delete: func(_ context.Context, _ string) error { return errArchiveRequired },
				update: func(_ context.Context, _ string, _ growthbook.FeatureRequest) (*growthbook.Feature, error) {
					return nil, errBoom
				},
			},
			err: errors.Wrap(errBoom, errArchiveFeature),
		},
		"DeleteFailsAfterArchive": {
			reason: "A delete that still fails after a successful archive is wrapped and returned.",
			client: &fakeClient{
				delete: func(_ context.Context, _ string) error { return errArchiveRequired },
				update: func(_ context.Context, _ string, _ growthbook.FeatureRequest) (*growthbook.Feature, error) {
					return remote(), nil
				},
			},
			err: errors.Wrap(errArchiveRequired, errDeleteFeature),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.client}
			_, err := e.Delete(context.Background(), newFeature())
			if diff := cmp.Diff(tc.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
