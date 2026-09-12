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

// Package environment reconciles core.growthbook.crossplane.io Environment
// resources against the GrowthBook /v1/environments API.
package environment

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

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/core/v1alpha1"
	apisv1alpha1 "github.com/jz-wilson/provider-growthbook/apis/v1alpha1"
	"github.com/jz-wilson/provider-growthbook/internal/clients/growthbook"
)

const (
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create GrowthBook client"

	errGetEnvironment    = "cannot get environment"
	errCreateEnvironment = "cannot create environment"
	errUpdateEnvironment = "cannot update environment"
	errDeleteEnvironment = "cannot delete environment"
)

// EnvironmentClient is the subset of the GrowthBook API the controller needs.
type EnvironmentClient interface {
	GetEnvironment(ctx context.Context, id string) (*growthbook.Environment, error)
	CreateEnvironment(ctx context.Context, req growthbook.EnvironmentRequest) (*growthbook.Environment, error)
	UpdateEnvironment(ctx context.Context, id string, req growthbook.EnvironmentRequest) (*growthbook.Environment, error)
	DeleteEnvironment(ctx context.Context, id string) error
}

var newClient = func(creds []byte) (EnvironmentClient, error) {
	return growthbook.NewFromSecret(creds)
}

// SetupGated adds a controller that reconciles Environment managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup Environment controller"))
		}
	}, v1alpha1.EnvironmentGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles Environment managed resources.
// The default initializers stay on: the environment id is user-chosen, so
// metadata.name is the right default external name.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.EnvironmentGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithTypedExternalConnector[*v1alpha1.Environment](&connector{
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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.EnvironmentList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.EnvironmentList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.EnvironmentGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Environment{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient from the ProviderConfig credentials.
type connector struct {
	kube        client.Client
	usage       *resource.ProviderConfigUsageTracker
	newClientFn func(creds []byte) (EnvironmentClient, error)
}

// Connect tracks ProviderConfig usage, resolves the credentials secret, and
// builds a GrowthBook client from it.
func (c *connector) Connect(ctx context.Context, cr *v1alpha1.Environment) (managed.TypedExternalClient[*v1alpha1.Environment], error) {
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

// An external client reconciles one Environment against the GrowthBook API.
type external struct {
	client EnvironmentClient
}

// Observe looks the environment up by its external name (the environment id).
func (e *external) Observe(ctx context.Context, cr *v1alpha1.Environment) (managed.ExternalObservation, error) {
	env, err := e.client.GetEnvironment(ctx, meta.GetExternalName(cr))
	if err != nil {
		if growthbook.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetEnvironment)
	}

	lateInit := lateInitialize(&cr.Spec.ForProvider, env)
	cr.Status.AtProvider = observation(env)
	cr.SetConditions(xpv2.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(cr.Spec.ForProvider, env),
		ResourceLateInitialized: lateInit,
	}, nil
}

// Create posts the environment under the external name as its id.
func (e *external) Create(ctx context.Context, cr *v1alpha1.Environment) (managed.ExternalCreation, error) {
	env, err := e.client.CreateEnvironment(ctx, createRequest(meta.GetExternalName(cr), cr.Spec.ForProvider, cr.Spec.InitProvider))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateEnvironment)
	}

	cr.Status.AtProvider = observation(env)

	return managed.ExternalCreation{}, nil
}

// Update pushes the mutable fields with PUT. Parent is create-only and is
// never sent.
func (e *external) Update(ctx context.Context, cr *v1alpha1.Environment) (managed.ExternalUpdate, error) {
	env, err := e.client.UpdateEnvironment(ctx, meta.GetExternalName(cr), updateRequest(cr.Spec.ForProvider))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateEnvironment)
	}

	cr.Status.AtProvider = observation(env)

	return managed.ExternalUpdate{}, nil
}

// Delete removes the environment. One that is already gone is a success.
func (e *external) Delete(ctx context.Context, cr *v1alpha1.Environment) (managed.ExternalDelete, error) {
	if err := e.client.DeleteEnvironment(ctx, meta.GetExternalName(cr)); err != nil && !growthbook.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteEnvironment)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op: the HTTP client holds no long-lived connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
