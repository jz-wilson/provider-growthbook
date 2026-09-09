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

func TestEnvironmentRoundTrip(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody EnvironmentRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = EnvironmentRequest{}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		switch {
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"environments": []Environment{
				{ID: "production", Description: "Prod", ToggleOnList: true, DefaultState: false, Projects: []string{}},
				{ID: "staging", Description: "Stage", Projects: []string{"prj_1"}},
			}})
		case r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"deletedId":"staging"}`))
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"environment": Environment{
				ID: gotBody.ID, Description: derefStr(gotBody.Description), Projects: []string{},
			}})
		}
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	ctx := context.Background()

	envs, err := c.ListEnvironments(ctx)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	if gotPath != "/api/v1/environments" || len(envs) != 2 || envs[1].ID != "staging" {
		t.Errorf("ListEnvironments() path=%q envs=%+v", gotPath, envs)
	}

	// GetEnvironment is a list-and-filter because the API has no GET by id.
	e, err := c.GetEnvironment(ctx, "staging")
	if err != nil || e.Description != "Stage" || len(e.Projects) != 1 {
		t.Errorf("GetEnvironment(staging) = %+v, err %v", e, err)
	}
	if _, err := c.GetEnvironment(ctx, "nope"); !IsNotFound(err) {
		t.Errorf("GetEnvironment(nope) err = %v, want not found", err)
	}

	created, err := c.CreateEnvironment(ctx, EnvironmentRequest{ID: "qa", Description: ptrStr("QA")})
	if err != nil || gotMethod != http.MethodPost || gotPath != "/api/v1/environments" || created.ID != "qa" {
		t.Errorf("CreateEnvironment() err=%v method=%q path=%q got=%+v", err, gotMethod, gotPath, created)
	}

	if _, err := c.UpdateEnvironment(ctx, "qa", EnvironmentRequest{Description: ptrStr("QA2")}); err != nil || gotMethod != http.MethodPut || gotPath != "/api/v1/environments/qa" {
		t.Errorf("UpdateEnvironment() err=%v method=%q path=%q", err, gotMethod, gotPath)
	}
	if gotBody.ID != "" {
		t.Errorf("UpdateEnvironment() must not send id in the body, got %q", gotBody.ID)
	}

	if err := c.DeleteEnvironment(ctx, "staging"); err != nil || gotMethod != http.MethodDelete || gotPath != "/api/v1/environments/staging" {
		t.Errorf("DeleteEnvironment() err=%v method=%q path=%q", err, gotMethod, gotPath)
	}
}

func TestEnvironmentRequestOmitsUnset(t *testing.T) {
	b, err := json.Marshal(EnvironmentRequest{ID: "qa"})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(`{"id":"qa"}`, string(b)); diff != "" {
		t.Errorf("unset fields must be omitted: -want +got\n%s", diff)
	}
}

func ptrStr(s string) *string { return &s }

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
