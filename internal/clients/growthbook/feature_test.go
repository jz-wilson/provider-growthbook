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

func TestFeatureRoundTrip(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody FeatureRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = FeatureRequest{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/features/ft_missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Could not find feature"}`))
		case r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"deletedId":"ft_1"}`))
		default:
			_ = json.NewEncoder(w).Encode(featureEnvelope{Feature: Feature{
				ID:           "ft_1",
				ValueType:    "boolean",
				DefaultValue: "true",
				Description:  derefStr(gotBody.Description),
				Revision:     &FeatureRevision{Version: 1},
				DateCreated:  "2026-01-01T00:00:00Z",
				DateUpdated:  "2026-01-01T00:00:00Z",
			}})
		}
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	ctx := context.Background()

	f, err := c.CreateFeature(ctx, FeatureRequest{ID: "ft_1", ValueType: "boolean", DefaultValue: "true"})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if gotAuth != "Bearer secret_1" || gotMethod != http.MethodPost || gotPath != "/api/v2/features" || f.ID != "ft_1" {
		t.Errorf("CreateFeature() auth=%q method=%q path=%q id=%q", gotAuth, gotMethod, gotPath, f.ID)
	}
	if f.Revision == nil || f.Revision.Version != 1 {
		t.Errorf("CreateFeature() revision = %+v, want version 1", f.Revision)
	}

	if _, err := c.UpdateFeature(ctx, "ft_1", FeatureRequest{Description: ptrStr("updated")}); err != nil {
		t.Fatalf("UpdateFeature() error = %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v2/features/ft_1" {
		t.Errorf("UpdateFeature() method=%q path=%q", gotMethod, gotPath)
	}
	if gotBody.ID != "" || gotBody.ValueType != "" {
		t.Errorf("UpdateFeature() must not send id or valueType (immutable): %+v", gotBody)
	}

	if _, err := c.GetFeature(ctx, "ft_missing"); !IsNotFound(err) {
		t.Errorf("GetFeature(missing) err = %v, want not found", err)
	}

	if err := c.DeleteFeature(ctx, "ft_1"); err != nil || gotMethod != http.MethodDelete {
		t.Errorf("DeleteFeature() err=%v method=%q", err, gotMethod)
	}
}

func TestFeatureCreateRequestEncodesEnvironments(t *testing.T) {
	req := FeatureRequest{
		ID:           "ft_env",
		ValueType:    "boolean",
		DefaultValue: "false",
		Environments: map[string]FeatureEnvironmentRequest{
			"production": {Enabled: featurePtrBool(true)},
			"staging":    {Enabled: featurePtrBool(false)},
		},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var back FeatureRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if diff := cmp.Diff(req, back); diff != "" {
		t.Errorf("FeatureRequest round trip -want +got:\n%s", diff)
	}
}

func featurePtrBool(b bool) *bool { return &b }
