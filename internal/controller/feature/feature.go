/*
Copyright 2025 The Crossplane Authors.

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

// Package feature reconciles feature.growthbook.crossplane.io Feature
// resources against the GrowthBook /v2/features API.
package feature

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/feature/v1alpha1"
	apisv1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/v1alpha1"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/clients/growthbook"
)

const (
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create GrowthBook client"

	errGetFeature     = "cannot get feature"
	errCreateFeature  = "cannot create feature"
	errUpdateFeature  = "cannot update feature"
	errDeleteFeature  = "cannot delete feature"
	errArchiveFeature = "cannot archive feature before delete"
	errFeatureRules   = "invalid feature rules"
)

// FeatureClient is the subset of the GrowthBook API the controller needs.
type FeatureClient interface {
	GetFeature(ctx context.Context, id string) (*growthbook.Feature, error)
	CreateFeature(ctx context.Context, req growthbook.FeatureRequest) (*growthbook.Feature, error)
	UpdateFeature(ctx context.Context, id string, req growthbook.FeatureRequest) (*growthbook.Feature, error)
	DeleteFeature(ctx context.Context, id string) error
}

var newClient = func(creds []byte) (FeatureClient, error) {
	return growthbook.NewFromSecret(creds)
}

// SetupGated adds a controller that reconciles Feature managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup Feature controller"))
		}
	}, v1alpha1.FeatureGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles Feature managed resources.
// The default initializers stay on: the feature key is user-chosen, so
// metadata.name is the right default external name.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.FeatureGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithTypedExternalConnector[*v1alpha1.Feature](&connector{
			kube:        mgr.GetClient(),
			usage:       resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newClientFn: newClient}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))), //nolint:staticcheck // TODO(jbw976) Crossplane needs to update to the new events API, see https://github.com/crossplane/crossplane/issues/7152
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.FeatureList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.FeatureList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.FeatureGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Feature{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient from the ProviderConfig credentials.
type connector struct {
	kube        client.Client
	usage       *resource.ProviderConfigUsageTracker
	newClientFn func(creds []byte) (FeatureClient, error)
}

// Connect tracks ProviderConfig usage, resolves the credentials secret, and
// builds a GrowthBook client from it.
func (c *connector) Connect(ctx context.Context, cr *v1alpha1.Feature) (managed.TypedExternalClient[*v1alpha1.Feature], error) {
	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	var cd apisv1alpha1.ProviderCredentials

	ref := cr.GetProviderConfigReference()

	switch ref.Kind {
	case "ProviderConfig":
		pc := &apisv1alpha1.ProviderConfig{}
		if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: cr.GetNamespace()}, pc); err != nil {
			return nil, errors.Wrap(err, errGetPC)
		}
		cd = pc.Spec.Credentials
	case "ClusterProviderConfig":
		cpc := &apisv1alpha1.ClusterProviderConfig{}
		if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc); err != nil {
			return nil, errors.Wrap(err, errGetCPC)
		}
		cd = cpc.Spec.Credentials
	default:
		return nil, errors.Errorf("unsupported provider config kind: %s", ref.Kind)
	}
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	gb, err := c.newClientFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: gb}, nil
}

// An external client reconciles one Feature against the GrowthBook API.
type external struct {
	client FeatureClient
}

// Observe looks the feature up by its external name (the feature key).
func (e *external) Observe(ctx context.Context, cr *v1alpha1.Feature) (managed.ExternalObservation, error) {
	f, err := e.client.GetFeature(ctx, meta.GetExternalName(cr))
	if err != nil {
		if growthbook.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetFeature)
	}

	lateInit := lateInitialize(&cr.Spec.ForProvider, cr.Spec.InitProvider, f)
	cr.Status.AtProvider = observation(f)
	cr.SetConditions(xpv2.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(cr.Spec.ForProvider, f),
		ResourceLateInitialized: lateInit,
	}, nil
}

// Create posts the feature under the external name as its key.
func (e *external) Create(ctx context.Context, cr *v1alpha1.Feature) (managed.ExternalCreation, error) {
	req, err := createRequest(meta.GetExternalName(cr), cr.Spec.ForProvider, cr.Spec.InitProvider)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errFeatureRules)
	}

	f, err := e.client.CreateFeature(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateFeature)
	}

	cr.Status.AtProvider = observation(f)

	return managed.ExternalCreation{}, nil
}

// Update pushes the mutable fields. valueType is create-only and is never
// sent. GrowthBook publishes the change immediately; there is no separate
// draft step for these fields.
func (e *external) Update(ctx context.Context, cr *v1alpha1.Feature) (managed.ExternalUpdate, error) {
	req, err := updateRequest(cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errFeatureRules)
	}

	f, err := e.client.UpdateFeature(ctx, meta.GetExternalName(cr), req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateFeature)
	}

	cr.Status.AtProvider = observation(f)

	return managed.ExternalUpdate{}, nil
}

// Delete removes the feature. One that is already gone is a success. When
// the organization requires archiving before delete ("REST API always
// bypasses approval requirements" disabled), GrowthBook returns a 403
// asking the caller to archive the feature first; this archives it and
// retries the delete once.
func (e *external) Delete(ctx context.Context, cr *v1alpha1.Feature) (managed.ExternalDelete, error) {
	id := meta.GetExternalName(cr)
	err := e.client.DeleteFeature(ctx, id)
	if err == nil || growthbook.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	if !growthbook.IsArchiveRequired(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteFeature)
	}

	archived := true
	if _, archiveErr := e.client.UpdateFeature(ctx, id, growthbook.FeatureRequest{Archived: &archived}); archiveErr != nil {
		return managed.ExternalDelete{}, errors.Wrap(archiveErr, errArchiveFeature)
	}

	if err := e.client.DeleteFeature(ctx, id); err != nil && !growthbook.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteFeature)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op: the HTTP client holds no long-lived connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
