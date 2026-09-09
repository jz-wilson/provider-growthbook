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
	"net/http"
	"net/url"
)

// FeatureEnvironment is a feature's per-environment state as returned by
// the API.
type FeatureEnvironment struct {
	Enabled bool `json:"enabled"`
}

// FeatureRevision reports a feature's current published revision.
type FeatureRevision struct {
	Version int `json:"version"`
}

// Feature is the GrowthBook feature object as returned by the v2 API.
// This client only models the fields the provider manages (value,
// per-environment enabled toggles, and metadata); rules and other v2
// fields are intentionally omitted and pass through unread.
type Feature struct {
	ID           string                        `json:"id"`
	Archived     bool                          `json:"archived,omitempty"`
	Description  string                        `json:"description,omitempty"`
	Owner        string                        `json:"owner,omitempty"`
	Project      string                        `json:"project,omitempty"`
	ValueType    string                        `json:"valueType"`
	DefaultValue string                        `json:"defaultValue"`
	Tags         []string                      `json:"tags,omitempty"`
	Environments map[string]FeatureEnvironment `json:"environments,omitempty"`
	DateCreated  string                        `json:"dateCreated,omitempty"`
	DateUpdated  string                        `json:"dateUpdated,omitempty"`
	Revision     *FeatureRevision              `json:"revision,omitempty"`
}

// FeatureEnvironmentRequest sets a feature's enabled state in one
// environment.
type FeatureEnvironmentRequest struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// FeatureRequest is the body for POST /v2/features and
// POST /v2/features/{id}. ID and ValueType are create-only: callers must
// leave them empty on update, which the JSON encoding then omits, since
// the update endpoint rejects a valueType field entirely.
type FeatureRequest struct {
	ID           string                               `json:"id,omitempty"`
	ValueType    string                               `json:"valueType,omitempty"`
	DefaultValue string                               `json:"defaultValue,omitempty"`
	Description  *string                              `json:"description,omitempty"`
	Project      *string                              `json:"project,omitempty"`
	Tags         []string                             `json:"tags,omitempty"`
	Archived     *bool                                `json:"archived,omitempty"`
	Owner        *string                              `json:"owner,omitempty"`
	Environments map[string]FeatureEnvironmentRequest `json:"environments,omitempty"`
}

type featureEnvelope struct {
	Feature Feature `json:"feature"`
}

// GetFeature fetches one feature by id (its key). A missing feature
// returns an APIError satisfying IsNotFound. The response also carries
// revision history under FeatureWithRevisionsV2, which decodes fine into
// featureEnvelope since the extra fields are simply ignored.
func (c *Client) GetFeature(ctx context.Context, id string) (*Feature, error) {
	var out featureEnvelope
	if err := c.do(ctx, http.MethodGet, "/v2/features/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Feature, nil
}

// CreateFeature creates a feature with a caller-chosen key.
func (c *Client) CreateFeature(ctx context.Context, req FeatureRequest) (*Feature, error) {
	var out featureEnvelope
	if err := c.do(ctx, http.MethodPost, "/v2/features", req, &out); err != nil {
		return nil, err
	}
	return &out.Feature, nil
}

// UpdateFeature applies a partial update and publishes it immediately:
// GrowthBook's v2 update endpoint has no separate draft/publish step for
// these fields, so this call takes effect as soon as it returns. The id
// travels in the path only, and valueType can never be changed after
// creation, so both are dropped from req if set.
func (c *Client) UpdateFeature(ctx context.Context, id string, req FeatureRequest) (*Feature, error) {
	req.ID = ""
	req.ValueType = ""
	var out featureEnvelope
	if err := c.do(ctx, http.MethodPost, "/v2/features/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out.Feature, nil
}

// DeleteFeature deletes a feature.
func (c *Client) DeleteFeature(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v2/features/"+url.PathEscape(id), nil, nil)
}
