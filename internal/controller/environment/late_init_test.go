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

package environment

import (
	"context"
	"testing"

	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

// TestInitProviderSurvivesLateInitialization walks the full reconcile
// sequence that a single-Observe test cannot catch: after Create, the first
// Observe must not late-initialize a field the user set only in
// spec.initProvider into spec.forProvider. Otherwise that field becomes
// enforced, and the next API-side change to it is reported as drift and
// reverted, which defeats initProvider's create-only contract.
func TestInitProviderSurvivesLateInitialization(t *testing.T) {
	cr := environment(withInitDescription("init desc"))

	apiDescription := "init desc"
	client := &fakeClient{get: func(_ context.Context, _ string) (*growthbook.Environment, error) {
		r := remote()
		r.Description = apiDescription
		return r, nil
	}}
	e := external{client: client}

	if _, err := e.Observe(context.Background(), cr); err != nil {
		t.Fatalf("first e.Observe(...): unexpected error %v", err)
	}
	if cr.Spec.ForProvider.Description != nil {
		t.Fatalf("late-initialization copied initProvider-only description into forProvider: %q", *cr.Spec.ForProvider.Description)
	}

	apiDescription = "changed in the GrowthBook UI"
	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("second e.Observe(...): unexpected error %v", err)
	}
	if !got.ResourceUpToDate {
		t.Errorf("an API-side change to an initProvider-only field must not be reported as drift")
	}
}
