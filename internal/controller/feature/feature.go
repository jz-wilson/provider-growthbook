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

package feature

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/jz-wilson/provider-growthbook/apis/feature/v1alpha1"
	apisv1alpha1 "github.com/jz-wilson/provider-growthbook/apis/v1alpha1"
)

const (
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// ErrNotImplemented is returned by every External operation on Feature. The
// GrowthBook client for this resource has not been implemented yet, so
// applying a Feature today would otherwise appear to succeed while never
// actually reconciling anything against the GrowthBook API.
var ErrNotImplemented = errors.New("Feature is not implemented yet in provider-growthbook")

// A stubService is a placeholder for the GrowthBook client this controller
// will eventually use. It intentionally does nothing yet.
type stubService struct {
	// creds are stored, not ignored, so that once a real client lands here
	// wiring credentials through is a small diff rather than a rewrite.
	creds []byte
}

var (
	newStubService = func(creds []byte) (*stubService, error) { return &stubService{creds: creds}, nil }
)

// SetupGated adds a controller that reconciles Feature managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup Feature controller"))
		}
	}, v1alpha1.FeatureGroupVersionKind)
	return nil
}

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.FeatureGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithTypedExternalConnector[*v1alpha1.Feature](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newStubService}),
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

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(creds []byte) (*stubService, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
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

	svc, err := c.newServiceFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// service is a placeholder for the future GrowthBook client. Every
	// method below returns ErrNotImplemented until it is wired up.
	service *stubService
}

// Observe always fails: Feature reconciliation is not implemented yet. This
// surfaces as Synced=False with ReconcileError on the managed resource,
// rather than silently reporting a healthy resource that was never checked
// against the GrowthBook API.
func (c *external) Observe(_ context.Context, _ *v1alpha1.Feature) (managed.ExternalObservation, error) {
	return managed.ExternalObservation{}, ErrNotImplemented
}

// Create always fails: see Observe.
func (c *external) Create(_ context.Context, _ *v1alpha1.Feature) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, ErrNotImplemented
}

// Update always fails: see Observe.
func (c *external) Update(_ context.Context, _ *v1alpha1.Feature) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, ErrNotImplemented
}

// Delete always fails: see Observe. Note this means a Feature object cannot
// be deleted through normal reconciliation while this controller is
// unimplemented; an operator who needs to remove one must manually strip its
// finalizer (kubectl patch ... --type=merge -p '{"metadata":{"finalizers":[]}}")
// rather than relying on Crossplane to clean it up, so the object is never
// silently orphaned by this controller pretending deletion succeeded.
func (c *external) Delete(_ context.Context, _ *v1alpha1.Feature) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, ErrNotImplemented
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
