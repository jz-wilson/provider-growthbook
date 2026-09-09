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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseCredentials(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    Credentials
		wantErr bool
	}{
		"JSONWithURL": {
			in:   `{"apiKey":"secret_1","apiUrl":"https://gb.example/api/"}`,
			want: Credentials{APIKey: "secret_1", APIURL: "https://gb.example/api"},
		},
		"JSONDefaultURL": {
			in:   `{"apiKey":"secret_1"}`,
			want: Credentials{APIKey: "secret_1", APIURL: DefaultAPIURL},
		},
		"BareKey": {
			in:   "secret_1\n",
			want: Credentials{APIKey: "secret_1", APIURL: DefaultAPIURL},
		},
		"Empty":      {in: "", wantErr: true},
		"MissingKey": {in: `{"apiUrl":"https://gb.example/api"}`, wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseCredentials([]byte(tc.in))
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseCredentials() err = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseCredentials() -want +got:\n%s", diff)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"Nil":            {err: nil, want: false},
		"Plain404":       {err: &APIError{StatusCode: 404, Message: "not found"}, want: true},
		"Wrapped404":     {err: fmt.Errorf("wrap: %w", &APIError{StatusCode: 404}), want: true},
		"LiveProject400": {err: &APIError{StatusCode: 400, Message: "Could not find project with that id"}, want: true},
		"Other400":       {err: &APIError{StatusCode: 400, Message: "name is required"}, want: false},
		"PlanLimit402":   {err: &APIError{StatusCode: 402, Message: "Your plan only supports 1 project"}, want: false},
		"NotAPIError":    {err: errors.New("could not find anything"), want: false},
		"ServerError":    {err: &APIError{StatusCode: 500, Message: "not found"}, want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsArchiveRequired(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"Nil": {err: nil, want: false},
		"ArchiveRequired403": {
			err:  &APIError{StatusCode: 403, Message: "Cannot delete a live feature via the REST API when 'REST API always bypasses approval requirements' is disabled. Archive the feature first, or enable the bypass setting in organization settings."},
			want: true,
		},
		"WrappedArchiveRequired403": {
			err:  fmt.Errorf("wrap: %w", &APIError{StatusCode: 403, Message: "Archive the feature first"}),
			want: true,
		},
		"Other403":     {err: &APIError{StatusCode: 403, Message: "not authorized"}, want: false},
		"NotForbidden": {err: &APIError{StatusCode: 400, Message: "Archive the feature first"}, want: false},
		"NotAPIError":  {err: errors.New("archive the feature first"), want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsArchiveRequired(tc.err); got != tc.want {
				t.Errorf("IsArchiveRequired(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestProjectRoundTrip(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody ProjectRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/prj_missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Could not find project"}`))
		case r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"deletedId":"prj_1"}`))
		default:
			_ = json.NewEncoder(w).Encode(projectEnvelope{Project: Project{ID: "prj_1", Name: gotBody.Name}})
		}
	}))
	defer srv.Close()

	c, err := NewFromSecret([]byte(`{"apiKey":"secret_1","apiUrl":"` + srv.URL + `/api"}`))
	if err != nil {
		t.Fatalf("NewFromSecret() error = %v", err)
	}
	ctx := context.Background()

	p, err := c.CreateProject(ctx, ProjectRequest{Name: "web"})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if gotAuth != "Bearer secret_1" || gotMethod != http.MethodPost || gotPath != "/api/v1/projects" || p.ID != "prj_1" {
		t.Errorf("CreateProject() auth=%q method=%q path=%q id=%q", gotAuth, gotMethod, gotPath, p.ID)
	}

	if _, err := c.UpdateProject(ctx, "prj_1", ProjectRequest{Name: "web2"}); err != nil || gotMethod != http.MethodPut || gotPath != "/api/v1/projects/prj_1" {
		t.Errorf("UpdateProject() err=%v method=%q path=%q", err, gotMethod, gotPath)
	}

	if _, err := c.GetProject(ctx, "prj_missing"); !IsNotFound(err) {
		t.Errorf("GetProject(missing) err = %v, want not found", err)
	}

	if err := c.DeleteProject(ctx, "prj_1"); err != nil || gotMethod != http.MethodDelete {
		t.Errorf("DeleteProject() err=%v method=%q", err, gotMethod)
	}
}
