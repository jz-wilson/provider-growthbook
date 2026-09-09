//go:build integration

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
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// Integration tests run against a real GrowthBook API. Start one with
// e2e/docker-compose.yml, mint a key with e2e/bootstrap.sh, then:
//
//	GROWTHBOOK_API_KEY=... GROWTHBOOK_API_URL=http://localhost:3100/api \
//	  go test -tags integration ./internal/clients/...
//
// A self-hosted GrowthBook without a license runs on the free plan, which
// allows exactly one project and no custom environments (the API answers
// 402). The tests are written against that baseline: they create when the
// plan allows it and otherwise assert the 402 surfaces as an APIError, and
// they exercise update paths on resources that already exist.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	key := os.Getenv("GROWTHBOOK_API_KEY")
	if key == "" {
		t.Skip("GROWTHBOOK_API_KEY not set")
	}
	c, err := New(Credentials{APIKey: key, APIURL: os.Getenv("GROWTHBOOK_API_URL")})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

func isPlanLimit(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == http.StatusPaymentRequired
}

func TestIntegrationProjectLifecycle(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("it-%d", time.Now().UnixNano())
	p, err := c.CreateProject(ctx, ProjectRequest{Name: name, Description: ptrStr("integration")})
	switch {
	case isPlanLimit(err):
		t.Logf("plan limit reached, exercising update on an existing project instead: %v", err)
		testProjectUpdateOnExisting(ctx, t, c)
		return
	case err != nil:
		t.Fatalf("CreateProject() error = %v", err)
	}
	t.Cleanup(func() { _ = c.DeleteProject(context.Background(), p.ID) })
	if p.ID == "" || p.Name != name {
		t.Fatalf("CreateProject() = %+v", p)
	}

	got, err := c.GetProject(ctx, p.ID)
	if err != nil || got.Description != "integration" {
		t.Fatalf("GetProject() = %+v, err %v", got, err)
	}

	upd, err := c.UpdateProject(ctx, p.ID, ProjectRequest{Description: ptrStr("updated")})
	if err != nil || upd.Description != "updated" || upd.Name != name {
		t.Fatalf("UpdateProject() = %+v, err %v", upd, err)
	}

	if err := c.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := c.GetProject(ctx, p.ID); !IsNotFound(err) {
		t.Fatalf("GetProject() after delete err = %v, want not found", err)
	}
}

// testProjectUpdateOnExisting updates and reverts the description of the
// first existing project, which the free plan always allows.
func testProjectUpdateOnExisting(ctx context.Context, t *testing.T, c *Client) {
	t.Helper()
	var list struct {
		Projects []Project `json:"projects"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/projects", nil, &list); err != nil || len(list.Projects) == 0 {
		t.Fatalf("list projects = %+v, err %v", list, err)
	}
	p := list.Projects[0]
	t.Cleanup(func() {
		_, _ = c.UpdateProject(context.Background(), p.ID, ProjectRequest{Description: ptrStr(p.Description)})
	})

	upd, err := c.UpdateProject(ctx, p.ID, ProjectRequest{Description: ptrStr("integration-updated")})
	if err != nil || upd.Description != "integration-updated" {
		t.Fatalf("UpdateProject(existing) = %+v, err %v", upd, err)
	}
	got, err := c.GetProject(ctx, p.ID)
	if err != nil || got.Description != "integration-updated" {
		t.Fatalf("GetProject(existing) = %+v, err %v", got, err)
	}
	if _, err := c.GetProject(ctx, "prj_does_not_exist"); !IsNotFound(err) {
		t.Fatalf("GetProject(missing) err = %v, want not found", err)
	}
}

func TestIntegrationEnvironmentLifecycle(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id := fmt.Sprintf("it_%d", time.Now().UnixNano())
	e, err := c.CreateEnvironment(ctx, EnvironmentRequest{ID: id, Description: ptrStr("integration"), ToggleOnList: ptrBool(true)})
	switch {
	case isPlanLimit(err):
		t.Logf("plan limit reached, exercising update on the default environment instead: %v", err)
		testEnvironmentUpdateOnDefault(ctx, t, c)
		return
	case err != nil:
		t.Fatalf("CreateEnvironment() error = %v", err)
	}
	t.Cleanup(func() { _ = c.DeleteEnvironment(context.Background(), id) })
	if e.ID != id {
		t.Fatalf("CreateEnvironment() = %+v", e)
	}

	got, err := c.GetEnvironment(ctx, id)
	if err != nil || got.Description != "integration" || !got.ToggleOnList {
		t.Fatalf("GetEnvironment() = %+v, err %v", got, err)
	}

	upd, err := c.UpdateEnvironment(ctx, id, EnvironmentRequest{Description: ptrStr("updated"), DefaultState: ptrBool(true)})
	if err != nil || upd.Description != "updated" || !upd.DefaultState {
		t.Fatalf("UpdateEnvironment() = %+v, err %v", upd, err)
	}

	if err := c.DeleteEnvironment(ctx, id); err != nil {
		t.Fatalf("DeleteEnvironment() error = %v", err)
	}
	if _, err := c.GetEnvironment(ctx, id); !IsNotFound(err) {
		t.Fatalf("GetEnvironment() after delete err = %v, want not found", err)
	}
}

// testEnvironmentUpdateOnDefault updates and reverts the description of the
// built-in production environment.
func testEnvironmentUpdateOnDefault(ctx context.Context, t *testing.T, c *Client) {
	t.Helper()
	const id = "production"
	orig, err := c.GetEnvironment(ctx, id)
	if err != nil {
		t.Fatalf("GetEnvironment(%s) error = %v", id, err)
	}
	t.Cleanup(func() {
		_, _ = c.UpdateEnvironment(context.Background(), id, EnvironmentRequest{Description: ptrStr(orig.Description)})
	})

	upd, err := c.UpdateEnvironment(ctx, id, EnvironmentRequest{Description: ptrStr("integration-updated")})
	if err != nil || upd.Description != "integration-updated" {
		t.Fatalf("UpdateEnvironment(%s) = %+v, err %v", id, upd, err)
	}
	got, err := c.GetEnvironment(ctx, id)
	if err != nil || got.Description != "integration-updated" {
		t.Fatalf("GetEnvironment(%s) = %+v, err %v", id, got, err)
	}
	if _, err := c.GetEnvironment(ctx, "does-not-exist"); !IsNotFound(err) {
		t.Fatalf("GetEnvironment(missing) err = %v, want not found", err)
	}
}

// --- Feature lifecycle (added for the Feature managed resource) ---

func TestIntegrationFeatureLifecycle(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id := fmt.Sprintf("it_feature_%d", time.Now().UnixNano())
	f, err := c.CreateFeature(ctx, FeatureRequest{ID: id, ValueType: "boolean", DefaultValue: "true"})
	switch {
	case isPlanLimit(err):
		t.Logf("plan limit reached, cannot create a feature: %v", err)
		t.Skip("plan does not allow creating features")
		return
	case err != nil:
		t.Fatalf("CreateFeature() error = %v", err)
	}
	t.Cleanup(func() { _ = deleteFeatureArchivingIfRequired(context.Background(), c, id) })
	if f.ID != id || f.ValueType != "boolean" || f.DefaultValue != "true" {
		t.Fatalf("CreateFeature() = %+v", f)
	}

	got, err := c.GetFeature(ctx, id)
	if err != nil || got.ID != id {
		t.Fatalf("GetFeature() = %+v, err %v", got, err)
	}

	upd, err := c.UpdateFeature(ctx, id, FeatureRequest{Description: ptrStr("integration-updated")})
	if err != nil || upd.Description != "integration-updated" {
		t.Fatalf("UpdateFeature() = %+v, err %v", upd, err)
	}

	// Some organizations disable "REST API always bypasses approval
	// requirements", in which case GrowthBook refuses to delete a live
	// feature via the REST API with a 403 asking the caller to archive it
	// first. Exercise the same archive-then-delete path the controller
	// uses.
	if err := deleteFeatureArchivingIfRequired(ctx, c, id); err != nil {
		t.Fatalf("deleteFeatureArchivingIfRequired() error = %v", err)
	}
	if _, err := c.GetFeature(ctx, id); !IsNotFound(err) {
		t.Fatalf("GetFeature() after delete err = %v, want not found", err)
	}
}

// deleteFeatureArchivingIfRequired mirrors the controller's Delete: if the
// API demands the feature be archived first, it archives and retries once.
func deleteFeatureArchivingIfRequired(ctx context.Context, c *Client, id string) error {
	err := c.DeleteFeature(ctx, id)
	if err == nil || IsNotFound(err) {
		return nil
	}
	if !IsArchiveRequired(err) {
		return err
	}
	if _, archiveErr := c.UpdateFeature(ctx, id, FeatureRequest{Archived: ptrBool(true)}); archiveErr != nil {
		return archiveErr
	}
	if err := c.DeleteFeature(ctx, id); err != nil && !IsNotFound(err) {
		return err
	}
	return nil
}

func ptrBool(b bool) *bool { return &b }
