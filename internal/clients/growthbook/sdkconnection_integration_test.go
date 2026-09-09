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
	"fmt"
	"testing"
	"time"
)

// TestIntegrationSDKConnectionLifecycle exercises the SDK connection CRUD
// path against a real GrowthBook instance. A self-hosted instance without a
// license may reject creates on the free plan (402); when that happens the
// test falls back to listing existing connections and exercising Get on one
// of them instead.
func TestIntegrationSDKConnectionLifecycle(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("it-sdk-%d", time.Now().UnixNano())
	sc, err := c.CreateSDKConnection(ctx, SDKConnectionRequest{Name: name, Language: "javascript", Environment: "production"})
	switch {
	case isPlanLimit(err):
		t.Logf("plan limit reached, exercising Get on an existing sdk connection instead: %v", err)
		testSDKConnectionGetExisting(ctx, t, c)
		return
	case err != nil:
		t.Fatalf("CreateSDKConnection() error = %v", err)
	}
	t.Cleanup(func() { _ = c.DeleteSDKConnection(context.Background(), sc.ID) })
	if sc.ID == "" || sc.Name != name || sc.Key == "" {
		t.Fatalf("CreateSDKConnection() = %+v", sc)
	}

	got, err := c.GetSDKConnection(ctx, sc.ID)
	if err != nil || got.Name != name {
		t.Fatalf("GetSDKConnection() = %+v, err %v", got, err)
	}

	upd, err := c.UpdateSDKConnection(ctx, sc.ID, SDKConnectionRequest{Name: name + "-updated"})
	if err != nil || upd.Name != name+"-updated" {
		t.Fatalf("UpdateSDKConnection() = %+v, err %v", upd, err)
	}

	if err := c.DeleteSDKConnection(ctx, sc.ID); err != nil {
		t.Fatalf("DeleteSDKConnection() error = %v", err)
	}
	if _, err := c.GetSDKConnection(ctx, sc.ID); !IsNotFound(err) {
		t.Fatalf("GetSDKConnection() after delete err = %v, want not found", err)
	}
}

// testSDKConnectionGetExisting asserts Get works against whatever SDK
// connection the instance already has, which the free plan always allows.
func testSDKConnectionGetExisting(ctx context.Context, t *testing.T, c *Client) {
	t.Helper()
	conns, err := c.ListSDKConnections(ctx)
	if err != nil || len(conns) == 0 {
		t.Fatalf("ListSDKConnections() = %+v, err %v", conns, err)
	}
	got, err := c.GetSDKConnection(ctx, conns[0].ID)
	if err != nil || got.ID != conns[0].ID {
		t.Fatalf("GetSDKConnection(existing) = %+v, err %v", got, err)
	}
	if _, err := c.GetSDKConnection(ctx, "sdk_does_not_exist"); !IsNotFound(err) {
		t.Fatalf("GetSDKConnection(missing) err = %v, want not found", err)
	}
}
