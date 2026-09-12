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

package sdkconnection

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	v1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/sdk/v1alpha1"
	apisv1alpha1 "github.com/jz-wilson/crossplane-provider-growthbook/apis/v1alpha1"
)

const (
	connectTestPCName = "pc"
	connectTestSecret = "creds"
	connectTestNS     = "ns"
	connectTestKey    = "token"
)

func withProviderConfigRef(kind, name string) func(*v1alpha1.SDKConnection) {
	return func(cr *v1alpha1.SDKConnection) {
		cr.SetNamespace(connectTestNS)
		cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: kind, Name: name})
	}
}

func secretCredentials() apisv1alpha1.ProviderCredentials {
	return apisv1alpha1.ProviderCredentials{
		Source: xpv2.CredentialsSourceSecret,
		CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
			SecretRef: &xpv2.SecretKeySelector{
				SecretReference: xpv2.SecretReference{Name: connectTestSecret, Namespace: connectTestNS},
				Key:             connectTestKey,
			},
		},
	}
}

func connectMockClient(pc *apisv1alpha1.ProviderConfig, secretData map[string][]byte) *test.MockClient {
	return &test.MockClient{
		MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
			switch o := obj.(type) {
			case *apisv1alpha1.ProviderConfig:
				if pc == nil || key.Name != pc.Name {
					return apierrors.NewNotFound(schema.GroupResource{Resource: "providerconfigs"}, key.Name)
				}
				*o = *pc
				return nil
			case *corev1.Secret:
				if secretData == nil || key.Name != connectTestSecret {
					return apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, key.Name)
				}
				o.Data = secretData
				return nil
			case *apisv1alpha1.ProviderConfigUsage:
				return apierrors.NewNotFound(schema.GroupResource{Resource: "providerconfigusages"}, key.Name)
			default:
				return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
			}
		},
		MockCreate: test.NewMockCreateFn(nil),
	}
}

func TestConnect(t *testing.T) {
	pc := &apisv1alpha1.ProviderConfig{}
	pc.SetName(connectTestPCName)
	pc.Spec.Credentials = secretCredentials()

	type args struct {
		kube        client.Client
		newClientFn func(creds []byte) (SDKConnectionClient, error)
		cr          *v1alpha1.SDKConnection
	}
	type want struct {
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"ProviderConfigResolvesClient": {
			args: args{
				kube: connectMockClient(pc, map[string][]byte{connectTestKey: []byte("secret-value")}),
				newClientFn: func(creds []byte) (SDKConnectionClient, error) {
					if string(creds) != "secret-value" {
						t.Fatalf("unexpected creds passed to newClientFn: %q", creds)
					}
					return &fakeClient{}, nil
				},
				cr: sdkConnection(connName, withProviderConfigRef("ProviderConfig", connectTestPCName)),
			},
			want: want{err: nil},
		},
		"UnsupportedProviderConfigKind": {
			args: args{
				kube:        connectMockClient(pc, map[string][]byte{connectTestKey: []byte("secret-value")}),
				newClientFn: func(creds []byte) (SDKConnectionClient, error) { return &fakeClient{}, nil },
				cr:          sdkConnection(connName, withProviderConfigRef("BogusConfig", "whatever")),
			},
			want: want{err: errors.Errorf("unsupported provider config kind: %s", "BogusConfig")},
		},
		"GetProviderConfigError": {
			args: args{
				kube:        connectMockClient(nil, nil),
				newClientFn: func(creds []byte) (SDKConnectionClient, error) { return &fakeClient{}, nil },
				cr:          sdkConnection(connName, withProviderConfigRef("ProviderConfig", connectTestPCName)),
			},
			want: want{err: errors.Wrap(apierrors.NewNotFound(schema.GroupResource{Resource: "providerconfigs"}, connectTestPCName), errGetPC)},
		},
		"NewClientError": {
			args: args{
				kube: connectMockClient(pc, map[string][]byte{connectTestKey: []byte("secret-value")}),
				newClientFn: func(creds []byte) (SDKConnectionClient, error) {
					return nil, errBoom
				},
				cr: sdkConnection(connName, withProviderConfigRef("ProviderConfig", connectTestPCName)),
			},
			want: want{err: errors.Wrap(errBoom, errNewClient)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &connector{
				kube:        tc.args.kube,
				usage:       resource.NewProviderConfigUsageTracker(tc.args.kube, &apisv1alpha1.ProviderConfigUsage{}),
				newClientFn: tc.args.newClientFn,
			}

			_, err := c.Connect(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Connect(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
