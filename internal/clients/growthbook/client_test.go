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
