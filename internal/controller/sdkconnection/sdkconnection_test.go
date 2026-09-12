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

package sdkconnection

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/sdk/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

// fakeClient implements SDKConnectionClient with pluggable behaviour.
type fakeClient struct {
	get    func(ctx context.Context, id string) (*growthbook.SDKConnection, error)
	create func(ctx context.Context, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error)
	update func(ctx context.Context, id string, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error)
	delete func(ctx context.Context, id string) error
}

func (f *fakeClient) GetSDKConnection(ctx context.Context, id string) (*growthbook.SDKConnection, error) {
	return f.get(ctx, id)
}

func (f *fakeClient) CreateSDKConnection(ctx context.Context, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error) {
	return f.create(ctx, req)
}

func (f *fakeClient) UpdateSDKConnection(ctx context.Context, id string, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error) {
	return f.update(ctx, id, req)
}

func (f *fakeClient) DeleteSDKConnection(ctx context.Context, id string) error {
	return f.delete(ctx, id)
}

var (
	errBoom     = errors.New("boom")
	errNotFound = &growthbook.APIError{StatusCode: http.StatusNotFound, Message: "Could not find sdkConnection"}
)

const connName = "web"

func sdkConnection(name string, mods ...func(*v1alpha1.SDKConnection)) *v1alpha1.SDKConnection {
	cr := &v1alpha1.SDKConnection{}
	cr.SetName(name)
	cr.Spec.ForProvider.Name = name
	cr.Spec.ForProvider.Language = langJavaScript
	cr.Spec.ForProvider.Environment = envProduction
	for _, m := range mods {
		m(cr)
	}
	return cr
}

func withExternalName(id string) func(*v1alpha1.SDKConnection) {
	return func(cr *v1alpha1.SDKConnection) { meta.SetExternalName(cr, id) }
}

func TestObserve(t *testing.T) {
	remote := &growthbook.SDKConnection{
		ID: sdkID, Name: connName, Languages: []string{langJavaScript}, Environment: envProduction,
		Key: "key_1", ProxySigningKey: "proxykey_1", ProxyHost: "https://proxy.example.com",
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		reason string
		client SDKConnectionClient
		cr     *v1alpha1.SDKConnection
		want   want
	}{
		"NoExternalName": {
			reason: "A resource without an external name has never been created.",
			client: &fakeClient{},
			cr:     sdkConnection(connName),
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"NotFound": {
			reason: "A 404 from the API means the connection is gone.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.SDKConnection, error) { return nil, errNotFound }},
			cr:     sdkConnection(connName, withExternalName(sdkID)),
			want:   want{o: managed.ExternalObservation{ResourceExists: false}},
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.SDKConnection, error) { return nil, errBoom }},
			cr:     sdkConnection(connName, withExternalName(sdkID)),
			want:   want{err: errors.Wrap(errBoom, errGetSDKConnection)},
		},
		"UpToDate": {
			reason: "A matching spec is up to date and its connection details are returned.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.SDKConnection, error) { return remote, nil }},
			cr:     sdkConnection(connName, withExternalName(sdkID)),
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
					ConnectionDetails: managed.ConnectionDetails{
						"key": []byte("key_1"), "proxySigningKey": []byte("proxykey_1"), "proxyHost": []byte("https://proxy.example.com"),
					},
				},
			},
		},
		"NameDrift": {
			reason: "A different name needs an update.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.SDKConnection, error) { return remote, nil }},
			cr: sdkConnection(connName, withExternalName(sdkID), func(cr *v1alpha1.SDKConnection) {
				cr.Spec.ForProvider.Name = "web-renamed"
			}),
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: true, ResourceUpToDate: false,
					ConnectionDetails: managed.ConnectionDetails{
						"key": []byte("key_1"), "proxySigningKey": []byte("proxykey_1"), "proxyHost": []byte("https://proxy.example.com"),
					},
				},
			},
		},
		"UnsetOptionalFieldsIgnored": {
			reason: "Optional fields the user never set do not cause drift even if GrowthBook reports a value.",
			client: &fakeClient{get: func(_ context.Context, _ string) (*growthbook.SDKConnection, error) {
				return &growthbook.SDKConnection{
					ID: sdkID, Name: connName, Languages: []string{langJavaScript}, Environment: envProduction,
					EncryptPayload: true, ProxyEnabled: true, ProxyHost: "https://proxy.example.com",
				}, nil
			}},
			cr: sdkConnection(connName, withExternalName(sdkID)),
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: true, ResourceUpToDate: true,
					ConnectionDetails: managed.ConnectionDetails{"proxyHost": []byte("https://proxy.example.com")},
				},
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
		})
	}
}

func TestCreate(t *testing.T) {
	var gotReq growthbook.SDKConnectionRequest
	client := &fakeClient{create: func(_ context.Context, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error) {
		gotReq = req
		return &growthbook.SDKConnection{
			ID: "sdk_new", Name: req.Name, Languages: []string{req.Language}, Environment: req.Environment,
			Key: "key_new", ProxySigningKey: "proxykey_new",
		}, nil
	}}
	cr := sdkConnection(connName)

	e := external{client: client}
	got, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("e.Create(...): unexpected error %v", err)
	}

	if id := meta.GetExternalName(cr); id != "sdk_new" {
		t.Errorf("external name = %q, want sdk_new", id)
	}
	if cr.Status.AtProvider.ID != "sdk_new" {
		t.Errorf("atProvider.id = %q, want sdk_new", cr.Status.AtProvider.ID)
	}
	wantReq := growthbook.SDKConnectionRequest{Name: connName, Language: langJavaScript, Environment: envProduction}
	if diff := cmp.Diff(wantReq, gotReq); diff != "" {
		t.Errorf("request body: -want, +got:\n%s", diff)
	}
	wantCD := managed.ConnectionDetails{"key": []byte("key_new"), "proxySigningKey": []byte("proxykey_new")}
	if diff := cmp.Diff(wantCD, got.ConnectionDetails); diff != "" {
		t.Errorf("connection details: -want, +got:\n%s", diff)
	}
}

func TestUpdate(t *testing.T) {
	var gotID string
	client := &fakeClient{update: func(_ context.Context, id string, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error) {
		gotID = id
		return &growthbook.SDKConnection{ID: id, Name: req.Name, DateUpdated: "2026-09-08T01:00:00Z", Key: "key_1"}, nil
	}}
	cr := sdkConnection(connName, withExternalName(sdkID))

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotID != sdkID {
		t.Errorf("updated id = %q, want sdk_1", gotID)
	}
	if cr.Status.AtProvider.DateUpdated != "2026-09-08T01:00:00Z" {
		t.Errorf("atProvider.dateUpdated not refreshed: %+v", cr.Status.AtProvider)
	}
}

func TestDelete(t *testing.T) {
	cases := map[string]struct {
		reason string
		client SDKConnectionClient
		cr     *v1alpha1.SDKConnection
		err    error
	}{
		"NoExternalName": {
			reason: "Nothing was ever created, so there is nothing to delete.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { t.Fatal("must not call API"); return nil }},
			cr:     sdkConnection(connName),
		},
		"AlreadyGone": {
			reason: "A 404 on delete is treated as success.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errNotFound }},
			cr:     sdkConnection(connName, withExternalName(sdkID)),
		},
		"APIError": {
			reason: "Other API errors are wrapped and returned.",
			client: &fakeClient{delete: func(_ context.Context, _ string) error { return errBoom }},
			cr:     sdkConnection(connName, withExternalName(sdkID)),
			err:    errors.Wrap(errBoom, errDeleteSDKConnection),
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

const (
	langJavaScript = "javascript"
	envProduction  = "production"
	sdkID          = "sdk_1"
)
