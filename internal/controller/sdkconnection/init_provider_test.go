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
	"testing"

	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/sdk/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

func ptr[T any](v T) *T { return &v }

func withInitSDKVersion(v string) func(*v1alpha1.SDKConnection) {
	return func(cr *v1alpha1.SDKConnection) { cr.Spec.InitProvider.SDKVersion = ptr(v) }
}

func withInitEncryptPayload(b bool) func(*v1alpha1.SDKConnection) {
	return func(cr *v1alpha1.SDKConnection) { cr.Spec.InitProvider.EncryptPayload = ptr(b) }
}

// TestCreateInitProvider covers the spec.initProvider merge semantics on
// Create: a field set only in initProvider is sent, and forProvider wins
// when both are set.
func TestCreateInitProvider(t *testing.T) {
	cases := map[string]struct {
		reason  string
		cr      *v1alpha1.SDKConnection
		wantReq growthbook.SDKConnectionRequest
	}{
		"InitProviderOnlyFieldIsSent": {
			reason: "A field set only in initProvider must be sent on Create.",
			cr:     sdkConnection(connName, withInitSDKVersion("0.30.0"), withInitEncryptPayload(true)),
			wantReq: growthbook.SDKConnectionRequest{
				Name: connName, Language: langJavaScript, Environment: envProduction,
				SDKVersion: ptr("0.30.0"), EncryptPayload: ptr(true),
			},
		},
		"ForProviderOverridesInitProvider": {
			reason: "forProvider wins over initProvider when both set the same field.",
			cr: sdkConnection(connName,
				withInitSDKVersion("0.30.0"),
				func(cr *v1alpha1.SDKConnection) { cr.Spec.ForProvider.SDKVersion = ptr("1.0.0") },
			),
			wantReq: growthbook.SDKConnectionRequest{
				Name: connName, Language: langJavaScript, Environment: envProduction, SDKVersion: ptr("1.0.0"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var gotReq growthbook.SDKConnectionRequest
			client := &fakeClient{create: func(_ context.Context, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error) {
				gotReq = req
				return &growthbook.SDKConnection{ID: "sdk_new", Name: req.Name, Languages: []string{req.Language}, Environment: req.Environment}, nil
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
// resource out of date.
func TestObserveIgnoresInitProviderDrift(t *testing.T) {
	cr := sdkConnection(connName, withInitEncryptPayload(true), withExternalName(sdkID))
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.SDKConnection, error) {
		return &growthbook.SDKConnection{
			ID: sdkID, Name: connName, Languages: []string{langJavaScript}, Environment: envProduction,
			EncryptPayload: false,
		}, nil
	}}

	e := external{client: client}
	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("e.Observe(...): unexpected error %v", err)
	}
	if !got.ResourceUpToDate {
		t.Errorf("initProvider-only field must never cause drift, got %+v", got)
	}
	if cr.Spec.InitProvider.EncryptPayload == nil || !*cr.Spec.InitProvider.EncryptPayload {
		t.Errorf("initProvider must remain untouched by Observe: got %v", cr.Spec.InitProvider.EncryptPayload)
	}
}

// TestUpdateNeverSendsInitProviderOnlyValues asserts Update never leaks
// initProvider-only field values into the PUT body.
func TestUpdateNeverSendsInitProviderOnlyValues(t *testing.T) {
	var gotReq growthbook.SDKConnectionRequest
	client := &fakeClient{update: func(_ context.Context, _ string, req growthbook.SDKConnectionRequest) (*growthbook.SDKConnection, error) {
		gotReq = req
		return &growthbook.SDKConnection{ID: sdkID, Name: req.Name}, nil
	}}
	cr := sdkConnection(connName, withInitSDKVersion("0.30.0"), withInitEncryptPayload(true), withExternalName(sdkID))

	e := external{client: client}
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatalf("e.Update(...): unexpected error %v", err)
	}
	if gotReq.SDKVersion != nil {
		t.Errorf("Update must not send initProvider-only sdkVersion: %+v", gotReq)
	}
	if gotReq.EncryptPayload != nil {
		t.Errorf("Update must not send initProvider-only encryptPayload: %+v", gotReq)
	}
}
