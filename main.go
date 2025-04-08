/*
Copyright 2021 CNCF.

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

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"

	peerpodcontrollers "github.com/confidential-containers/cloud-api-adaptor/src/peerpod-ctrl/controllers"
	peerpodconfigcontrollers "github.com/confidential-containers/cloud-api-adaptor/src/peerpodconfig-ctrl/controllers"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	peerpod "github.com/confidential-containers/cloud-api-adaptor/src/peerpod-ctrl/api/v1alpha1"
	peerpodconfig "github.com/confidential-containers/cloud-api-adaptor/src/peerpodconfig-ctrl/api/v1alpha1"

	ccv1beta1 "github.com/confidential-containers/operator/api/v1beta1"
	"github.com/confidential-containers/operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ccv1beta1.AddToScheme(scheme))

	utilruntime.Must(peerpodconfig.AddToScheme(scheme))

	utilruntime.Must(peerpod.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var secureMetrics bool
	var enableLeaderElection bool
	var probeAddr string
	var ccRuntimeNamespace string
	var enablePeerPodControllers bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"Enable role based authentication/authorization for the metrics endpoint")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&ccRuntimeNamespace, "cc-runtime-namespace", metav1.NamespaceSystem, "The namespace where CcRuntime secondary resources are created")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enablePeerPodControllers, "peer-pods", false,
		"Enable Peerpod controllers.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// TODO: add enable-http2 boolean flag which is what the latest
	// scaffolding gives to control http/2 enablement.
	disableHTTP2 := func(cfg *tls.Config) {
		cfg.NextProtos = []string{"http/1.1"}
	}

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts: []func(cfg *tls.Config){
			disableHTTP2,
		},
	}

	if secureMetrics {
		// More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "69bf4d38.confidentialcontainers.org",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ns := os.Getenv("CCRUNTIME_NAMESPACE")
	if ns == "" {
		ns = ccRuntimeNamespace
	}

	err = labelNamespace(context.TODO(), mgr, ns)
	if err != nil {
		setupLog.Error(err, "unable to add labels to namespace")
		os.Exit(1)
	}

	if err = (&controllers.CcRuntimeReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Namespace: ns,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CcRuntime")
		os.Exit(1)
	}

	if enablePeerPodControllers {
		if err = (&peerpodconfigcontrollers.PeerPodConfigReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("RemotePodConfig"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create peerpod config controller", "controller", "PeerPodConfig")
			os.Exit(1)
		}

		if err = (&peerpodcontrollers.PeerPodReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			// setting nil will delegate Provider creation to reconcile time.
			Provider: nil,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create peerpod resources controller", "controller", "PeerPod")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func labelNamespace(ctx context.Context, mgr manager.Manager, nsName string) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	err := mgr.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(ns), ns)
	if err != nil {
		setupLog.Error(err, "Unable to add label to the namespace")
		return err
	}

	setupLog.Info("Labelling Namespace")
	setupLog.Info("Labels: ", "Labels", ns.Labels)
	// Add namespace label to allow privilege pods via Pod Security Admission controller
	ns.Labels["pod-security.kubernetes.io/enforce"] = "privileged"
	ns.Labels["pod-security.kubernetes.io/audit"] = "privileged"
	ns.Labels["pod-security.kubernetes.io/warn"] = "privileged"

	return mgr.GetClient().Update(ctx, ns)
}
