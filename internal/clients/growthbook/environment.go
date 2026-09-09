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

// Environment is the GrowthBook environment object as returned by the API.
type Environment struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	ToggleOnList bool     `json:"toggleOnList"`
	DefaultState bool     `json:"defaultState"`
	Projects     []string `json:"projects"`
	Parent       string   `json:"parent,omitempty"`
}

// EnvironmentRequest is the body for POST /v1/environments and
// PUT /v1/environments/{id}. ID and Parent are create-only: callers must
// leave them empty on update, which the JSON encoding then omits.
type EnvironmentRequest struct {
	ID           string   `json:"id,omitempty"`
	Description  *string  `json:"description,omitempty"`
	ToggleOnList *bool    `json:"toggleOnList,omitempty"`
	DefaultState *bool    `json:"defaultState,omitempty"`
	Projects     []string `json:"projects,omitempty"`
	Parent       *string  `json:"parent,omitempty"`
}

type environmentEnvelope struct {
	Environment Environment `json:"environment"`
}

type environmentListEnvelope struct {
	Environments []Environment `json:"environments"`
}

// ListEnvironments returns every environment in the organization.
func (c *Client) ListEnvironments(ctx context.Context) ([]Environment, error) {
	var out environmentListEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/environments", nil, &out); err != nil {
		return nil, err
	}
	return out.Environments, nil
}

// GetEnvironment finds one environment by id. The API has no GET by id, so
// this lists and filters. A missing environment returns an APIError
// satisfying IsNotFound, matching the other resources.
func (c *Client) GetEnvironment(ctx context.Context, id string) (*Environment, error) {
	envs, err := c.ListEnvironments(ctx)
	if err != nil {
		return nil, err
	}
	for i := range envs {
		if envs[i].ID == id {
			return &envs[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "environment " + id + " not found"}
}

// CreateEnvironment creates an environment with a caller-chosen id.
func (c *Client) CreateEnvironment(ctx context.Context, req EnvironmentRequest) (*Environment, error) {
	var out environmentEnvelope
	if err := c.do(ctx, http.MethodPost, "/v1/environments", req, &out); err != nil {
		return nil, err
	}
	return &out.Environment, nil
}

// UpdateEnvironment applies a partial update. The id travels in the path
// only; any ID or Parent set on req is dropped because the API rejects them.
func (c *Client) UpdateEnvironment(ctx context.Context, id string, req EnvironmentRequest) (*Environment, error) {
	req.ID = ""
	req.Parent = nil
	var out environmentEnvelope
	if err := c.do(ctx, http.MethodPut, "/v1/environments/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out.Environment, nil
}

// DeleteEnvironment deletes an environment.
func (c *Client) DeleteEnvironment(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/environments/"+url.PathEscape(id), nil, nil)
}
