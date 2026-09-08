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

// ProjectSettings are per-project statistics settings that override the
// organization defaults. Nil pointers mean "not set".
type ProjectSettings struct {
	StatsEngine     *string  `json:"statsEngine,omitempty"`
	ConfidenceLevel *float64 `json:"confidenceLevel,omitempty"`
	PValueThreshold *float64 `json:"pValueThreshold,omitempty"`
}

// Project is the GrowthBook project object as returned by the API.
type Project struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Description    string           `json:"description,omitempty"`
	PublicID       string           `json:"publicId,omitempty"`
	RestrictAccess *bool            `json:"restrictAccess,omitempty"`
	Settings       *ProjectSettings `json:"settings,omitempty"`
	DateCreated    string           `json:"dateCreated,omitempty"`
	DateUpdated    string           `json:"dateUpdated,omitempty"`
}

// ProjectRequest is the body for POST /v1/projects and PUT /v1/projects/{id}.
// Name is required on create; every field is optional on update.
type ProjectRequest struct {
	Name           string           `json:"name,omitempty"`
	Description    *string          `json:"description,omitempty"`
	PublicID       *string          `json:"publicId,omitempty"`
	RestrictAccess *bool            `json:"restrictAccess,omitempty"`
	Settings       *ProjectSettings `json:"settings,omitempty"`
}

type projectEnvelope struct {
	Project Project `json:"project"`
}

// GetProject fetches one project by id. A missing project returns an
// APIError satisfying IsNotFound.
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var out projectEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/projects/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Project, nil
}

// CreateProject creates a project and returns the stored object.
func (c *Client) CreateProject(ctx context.Context, req ProjectRequest) (*Project, error) {
	var out projectEnvelope
	if err := c.do(ctx, http.MethodPost, "/v1/projects", req, &out); err != nil {
		return nil, err
	}
	return &out.Project, nil
}

// UpdateProject applies a partial update and returns the stored object.
func (c *Client) UpdateProject(ctx context.Context, id string, req ProjectRequest) (*Project, error) {
	var out projectEnvelope
	if err := c.do(ctx, http.MethodPut, "/v1/projects/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out.Project, nil
}

// DeleteProject deletes a project. Deleting a project that no longer exists
// returns an APIError satisfying IsNotFound.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/projects/"+url.PathEscape(id), nil, nil)
}
