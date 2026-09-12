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

// Package project reconciles core.growthbook.crossplane.io Project resources
// against the GrowthBook /v1/projects API.
package project

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

	errGetProject    = "cannot get project"
	errCreateProject = "cannot create project"
	errUpdateProject = "cannot update project"
	errDeleteProject = "cannot delete project"
	errSettings      = "invalid project settings"
)

// ProjectClient is the subset of the GrowthBook API the controller needs.
// It exists so tests can substitute a fake.
type ProjectClient interface {
	GetProject(ctx context.Context, id string) (*growthbook.Project, error)
	CreateProject(ctx context.Context, req growthbook.ProjectRequest) (*growthbook.Project, error)
	UpdateProject(ctx context.Context, id string, req growthbook.ProjectRequest) (*growthbook.Project, error)
	DeleteProject(ctx context.Context, id string) error
}

var newClient = func(creds []byte) (ProjectClient, error) {
	return growthbook.NewFromSecret(creds)
}

// SetupGated adds a controller that reconciles Project managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup Project controller"))
		}
	}, v1alpha1.ProjectGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles Project managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ProjectGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithTypedExternalConnector[*v1alpha1.Project](&connector{
			kube:        mgr.GetClient(),
			usage:       resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newClientFn: newClient}),
		// GrowthBook assigns project ids, so the external name must stay
		// empty until Create sets it. This drops the default initializer
		// that copies metadata.name into the external-name annotation.
		managed.WithInitializers(),
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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.ProjectList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ProjectList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ProjectGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Project{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector produces an ExternalClient from the ProviderConfig credentials.
type connector struct {
	kube        client.Client
	usage       *resource.ProviderConfigUsageTracker
	newClientFn func(creds []byte) (ProjectClient, error)
}

// Connect tracks ProviderConfig usage, resolves the credentials secret, and
// builds a GrowthBook client from it.
func (c *connector) Connect(ctx context.Context, cr *v1alpha1.Project) (managed.TypedExternalClient[*v1alpha1.Project], error) {
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

// An external client reconciles one Project against the GrowthBook API.
type external struct {
	client ProjectClient
}

// Observe looks the project up by its external name (the GrowthBook id).
func (e *external) Observe(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	p, err := e.client.GetProject(ctx, id)
	if err != nil {
		if growthbook.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetProject)
	}

	lateInit := lateInitialize(&cr.Spec.ForProvider, cr.Spec.InitProvider, p)
	cr.Status.AtProvider = observation(p)
	cr.SetConditions(xpv2.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(cr.Spec.ForProvider, p),
		ResourceLateInitialized: lateInit,
	}, nil
}

// Create posts the project and records the returned id as the external name.
func (e *external) Create(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalCreation, error) {
	req, err := createRequest(cr.Spec.ForProvider, cr.Spec.InitProvider)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSettings)
	}

	p, err := e.client.CreateProject(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateProject)
	}

	meta.SetExternalName(cr, p.ID)
	cr.Status.AtProvider = observation(p)

	return managed.ExternalCreation{}, nil
}

// Update pushes the full desired spec with PUT.
func (e *external) Update(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalUpdate, error) {
	req, err := request(cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errSettings)
	}

	p, err := e.client.UpdateProject(ctx, meta.GetExternalName(cr), req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateProject)
	}

	cr.Status.AtProvider = observation(p)

	return managed.ExternalUpdate{}, nil
}

// Delete removes the project. A project that is already gone is a success.
func (e *external) Delete(ctx context.Context, cr *v1alpha1.Project) (managed.ExternalDelete, error) {
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteProject(ctx, id); err != nil && !growthbook.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteProject)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op: the HTTP client holds no long-lived connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
