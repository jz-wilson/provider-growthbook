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

// SDKConnection is the GrowthBook SDK Connection object as returned by the
// API. Note the API asymmetry: the create/update request takes a single
// "language" string, but every response (including the one returned by
// POST/PUT) reports the set as "languages" (a slice).
type SDKConnection struct {
	ID                                  string   `json:"id"`
	DateCreated                         string   `json:"dateCreated"`
	DateUpdated                         string   `json:"dateUpdated"`
	Name                                string   `json:"name"`
	Organization                        string   `json:"organization"`
	Languages                           []string `json:"languages"`
	SDKVersion                          string   `json:"sdkVersion,omitempty"`
	Environment                         string   `json:"environment"`
	Project                             string   `json:"project"`
	Projects                            []string `json:"projects,omitempty"`
	EncryptPayload                      bool     `json:"encryptPayload"`
	EncryptionKey                       string   `json:"encryptionKey"`
	IncludeVisualExperiments            *bool    `json:"includeVisualExperiments,omitempty"`
	IncludeDraftExperiments             *bool    `json:"includeDraftExperiments,omitempty"`
	IncludeDraftExperimentRefs          *bool    `json:"includeDraftExperimentRefs,omitempty"`
	IncludeExperimentNames              *bool    `json:"includeExperimentNames,omitempty"`
	IncludeRedirectExperiments          *bool    `json:"includeRedirectExperiments,omitempty"`
	IncludeRuleIds                      *bool    `json:"includeRuleIds,omitempty"`
	IncludeProjectIdInMetadata          *bool    `json:"includeProjectIdInMetadata,omitempty"`
	IncludeCustomFieldsInMetadata       *bool    `json:"includeCustomFieldsInMetadata,omitempty"`
	AllowedCustomFieldsInMetadata       []string `json:"allowedCustomFieldsInMetadata,omitempty"`
	IncludeTagsInMetadata               *bool    `json:"includeTagsInMetadata,omitempty"`
	IncludeExperimentScheduleInMetadata *bool    `json:"includeExperimentScheduleInMetadata,omitempty"`
	Key                                 string   `json:"key"`
	ProxyEnabled                        bool     `json:"proxyEnabled"`
	ProxyHost                           string   `json:"proxyHost"`
	ProxySigningKey                     string   `json:"proxySigningKey"`
	SSEEnabled                          *bool    `json:"sseEnabled,omitempty"`
	HashSecureAttributes                *bool    `json:"hashSecureAttributes,omitempty"`
	RemoteEvalEnabled                   *bool    `json:"remoteEvalEnabled,omitempty"`
	SavedGroupReferencesEnabled         *bool    `json:"savedGroupReferencesEnabled,omitempty"`
	IncludeReferencedPrerequisites      *bool    `json:"includeReferencedPrerequisites,omitempty"`
	Connected                           *bool    `json:"connected,omitempty"`
}

// SDKConnectionRequest is the body for POST /v1/sdk-connections and
// PUT /v1/sdk-connections/{id}. Name, Language, and Environment are required
// on create; every field is optional on update.
type SDKConnectionRequest struct {
	Name                                string   `json:"name,omitempty"`
	Language                            string   `json:"language,omitempty"`
	Environment                         string   `json:"environment,omitempty"`
	SDKVersion                          *string  `json:"sdkVersion,omitempty"`
	Projects                            []string `json:"projects,omitempty"`
	EncryptPayload                      *bool    `json:"encryptPayload,omitempty"`
	IncludeVisualExperiments            *bool    `json:"includeVisualExperiments,omitempty"`
	IncludeDraftExperiments             *bool    `json:"includeDraftExperiments,omitempty"`
	IncludeDraftExperimentRefs          *bool    `json:"includeDraftExperimentRefs,omitempty"`
	IncludeExperimentNames              *bool    `json:"includeExperimentNames,omitempty"`
	IncludeRedirectExperiments          *bool    `json:"includeRedirectExperiments,omitempty"`
	IncludeRuleIds                      *bool    `json:"includeRuleIds,omitempty"`
	IncludeProjectIdInMetadata          *bool    `json:"includeProjectIdInMetadata,omitempty"`
	IncludeCustomFieldsInMetadata       *bool    `json:"includeCustomFieldsInMetadata,omitempty"`
	AllowedCustomFieldsInMetadata       []string `json:"allowedCustomFieldsInMetadata,omitempty"`
	IncludeTagsInMetadata               *bool    `json:"includeTagsInMetadata,omitempty"`
	IncludeExperimentScheduleInMetadata *bool    `json:"includeExperimentScheduleInMetadata,omitempty"`
	ProxyEnabled                        *bool    `json:"proxyEnabled,omitempty"`
	ProxyHost                           *string  `json:"proxyHost,omitempty"`
	HashSecureAttributes                *bool    `json:"hashSecureAttributes,omitempty"`
	RemoteEvalEnabled                   *bool    `json:"remoteEvalEnabled,omitempty"`
	SavedGroupReferencesEnabled         *bool    `json:"savedGroupReferencesEnabled,omitempty"`
	IncludeReferencedPrerequisites      *bool    `json:"includeReferencedPrerequisites,omitempty"`
}

type sdkConnectionEnvelope struct {
	SDKConnection SDKConnection `json:"sdkConnection"`
}

type sdkConnectionListEnvelope struct {
	Connections []SDKConnection `json:"connections"`
}

// GetSDKConnection fetches one SDK connection by id. A missing connection
// returns an APIError satisfying IsNotFound.
func (c *Client) GetSDKConnection(ctx context.Context, id string) (*SDKConnection, error) {
	var out sdkConnectionEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/sdk-connections/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out.SDKConnection, nil
}

// ListSDKConnections returns every SDK connection in the organization.
func (c *Client) ListSDKConnections(ctx context.Context) ([]SDKConnection, error) {
	var out sdkConnectionListEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/sdk-connections", nil, &out); err != nil {
		return nil, err
	}
	return out.Connections, nil
}

// CreateSDKConnection creates an SDK connection and returns the stored
// object, including the generated client key.
func (c *Client) CreateSDKConnection(ctx context.Context, req SDKConnectionRequest) (*SDKConnection, error) {
	var out sdkConnectionEnvelope
	if err := c.do(ctx, http.MethodPost, "/v1/sdk-connections", req, &out); err != nil {
		return nil, err
	}
	return &out.SDKConnection, nil
}

// UpdateSDKConnection applies a partial update and returns the stored
// object.
func (c *Client) UpdateSDKConnection(ctx context.Context, id string, req SDKConnectionRequest) (*SDKConnection, error) {
	var out sdkConnectionEnvelope
	if err := c.do(ctx, http.MethodPut, "/v1/sdk-connections/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out.SDKConnection, nil
}

// DeleteSDKConnection deletes an SDK connection. Deleting a connection that
// no longer exists returns an APIError satisfying IsNotFound.
func (c *Client) DeleteSDKConnection(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/sdk-connections/"+url.PathEscape(id), nil, nil)
}
