/*
Copyright 2020 The Crossplane Authors.

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

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/jz-wilson/crossplane-provider-growthbook/internal/controller/config"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/controller/environment"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/controller/feature"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/controller/project"
	"github.com/jz-wilson/crossplane-provider-growthbook/internal/controller/sdkconnection"
)

// SetupGated creates all GrowthBook controllers with safe-start support and adds them to
// the supplied manager.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		project.SetupGated,
		environment.SetupGated,
		feature.SetupGated,
		sdkconnection.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// Setup creates all GrowthBook controllers without safe-start gating and adds
// them to the supplied manager. Used when the provider's service account
// cannot watch CustomResourceDefinitions, for example on Crossplane
// installations that do not grant that permission.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		project.Setup,
		environment.Setup,
		feature.Setup,
		sdkconnection.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
