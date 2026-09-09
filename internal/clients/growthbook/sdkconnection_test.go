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

package growthbook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	testSDKConnName    = "web"
	testSDKLanguage    = "javascript"
	testSDKEnvironment = "production"
	testSDKKey         = "key_1"
	testSDKID          = "sdk_1"
)

func TestSDKConnectionRoundTrip(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody SDKConnectionRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = SDKConnectionRequest{}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sdk-connections":
			_ = json.NewEncoder(w).Encode(map[string]any{"connections": []SDKConnection{
				{ID: testSDKID, Name: testSDKConnName, Languages: []string{testSDKLanguage}, Environment: testSDKEnvironment, Key: testSDKKey},
			}})
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"sdkConnection": SDKConnection{
				ID: testSDKID, Name: testSDKConnName, Languages: []string{testSDKLanguage}, Environment: testSDKEnvironment, Key: testSDKKey,
			}})
		case r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"deletedId":testSDKID}`))
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"sdkConnection": SDKConnection{
				ID: testSDKID, Name: gotBody.Name, Languages: []string{gotBody.Language}, Environment: gotBody.Environment,
				Key: testSDKKey, ProxySigningKey: "proxykey_1", ProxyHost: "https://proxy.example.com",
			}})
		}
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	ctx := context.Background()

	conns, err := c.ListSDKConnections(ctx)
	if err != nil || gotPath != "/api/v1/sdk-connections" || len(conns) != 1 || conns[0].ID != testSDKID {
		t.Errorf("ListSDKConnections() = %+v, path=%q, err %v", conns, gotPath, err)
	}

	got, err := c.GetSDKConnection(ctx, testSDKID)
	if err != nil || gotPath != "/api/v1/sdk-connections/sdk_1" || got.Name != testSDKConnName || got.Key != testSDKKey {
		t.Errorf("GetSDKConnection() = %+v, path=%q, err %v", got, gotPath, err)
	}

	created, err := c.CreateSDKConnection(ctx, SDKConnectionRequest{Name: testSDKConnName, Language: testSDKLanguage, Environment: testSDKEnvironment})
	if err != nil || gotMethod != http.MethodPost || gotPath != "/api/v1/sdk-connections" {
		t.Fatalf("CreateSDKConnection() err=%v method=%q path=%q", err, gotMethod, gotPath)
	}
	if created.ID != testSDKID || created.Key != testSDKKey || created.ProxySigningKey != "proxykey_1" || created.ProxyHost != "https://proxy.example.com" {
		t.Errorf("CreateSDKConnection() = %+v, want connection-detail fields populated", created)
	}
	if len(created.Languages) != 1 || created.Languages[0] != "javascript" {
		t.Errorf("CreateSDKConnection() languages = %+v, want [javascript] (response reports 'languages' even though the request sent 'language')", created.Languages)
	}

	updated, err := c.UpdateSDKConnection(ctx, testSDKID, SDKConnectionRequest{Name: "web-renamed"})
	if err != nil || gotMethod != http.MethodPut || gotPath != "/api/v1/sdk-connections/sdk_1" || updated.Name != "web-renamed" {
		t.Errorf("UpdateSDKConnection() = %+v, method=%q, path=%q, err %v", updated, gotMethod, gotPath, err)
	}

	if err := c.DeleteSDKConnection(ctx, testSDKID); err != nil || gotMethod != http.MethodDelete || gotPath != "/api/v1/sdk-connections/sdk_1" {
		t.Errorf("DeleteSDKConnection() err=%v method=%q path=%q", err, gotMethod, gotPath)
	}
}

func TestSDKConnectionRequestOmitsUnset(t *testing.T) {
	b, err := json.Marshal(SDKConnectionRequest{Name: testSDKConnName, Language: testSDKLanguage, Environment: testSDKEnvironment})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(`{"name":"web","language":"javascript","environment":"production"}`, string(b)); diff != "" {
		t.Errorf("unset fields must be omitted: -want +got\n%s", diff)
	}
}

func TestSDKConnectionNotFound(t *testing.T) {
	// GrowthBook sometimes returns 400 with a "Could not find ..." message
	// for a missing SDK connection instead of a clean 404; IsNotFound
	// already handles both shapes for every resource.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Could not find sdkConnection with id sdk_missing"})
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}

	if _, err := c.GetSDKConnection(context.Background(), "sdk_missing"); !IsNotFound(err) {
		t.Errorf("GetSDKConnection(missing) err = %v, want not found", err)
	}
}
