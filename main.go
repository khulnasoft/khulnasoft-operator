/*
Copyright 2022.

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
	"flag"
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/controllers/khulnasoft/khulnasoftstarboard"
	"github.com/khulnasoft/khulnasoft-operator/controllers/ocp"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftcloudconnector"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftcsp"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftdatabase"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftenforcer"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftgateway"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftkubeenforcer"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftlightning"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftscanner"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftserver"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	version2 "github.com/khulnasoft/khulnasoft-operator/pkg/version"
	routev1 "github.com/openshift/api/route/v1"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"runtime"

	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	khulnasoftv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/khulnasoft/v1alpha1"
	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
)

var (
	scheme   = k8sRuntime.NewScheme()
	setupLog = logf.Log.WithName("setup")
)

const controllerMessage = "Unable to create controller"

func printVersion() {
	setupLog.Info(fmt.Sprintf("Operator Version: %s", version2.Version))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(khulnasoftv1alpha1.AddToScheme(scheme))

	isOpenshift, _ := ocp.VerifyRouteAPI()
	if isOpenshift {
		utilruntime.Must(routev1.AddToScheme(scheme))
	}
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// Parsing flags
	flag.Parse()

	encoderConfig := uzap.NewProductionEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	logf.SetLogger(zap.New(
		zap.Encoder(encoder),
		zap.Level(zapcore.InfoLevel),
		zap.StacktraceLevel(zapcore.PanicLevel),
	),
	)
	printVersion()

	ws := webhook.NewServer(webhook.Options{
		Port: 9443,
	})

	watchNamespace := extra.GetCurrentNameSpace()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr},
		WebhookServer:          ws,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "khulnasoft-operator-lock",
		Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
			watchNamespace: {},
		}},
	})

	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	if err = (&khulnasoftcsp.KhulnasoftCspReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftCsp")
		os.Exit(1)
	}

	if err = (&khulnasoftdatabase.KhulnasoftDatabaseReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftDatabase")
		os.Exit(1)
	}

	if err = (&khulnasoftenforcer.KhulnasoftEnforcerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftEnforcer")
		os.Exit(1)
	}

	if err = (&khulnasoftgateway.KhulnasoftGatewayReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftGateway")
		os.Exit(1)
	}

	if err = (&khulnasoftkubeenforcer.KhulnasoftKubeEnforcerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Certs:  khulnasoftkubeenforcer.GetKECerts(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftKubeEnforcer")
		os.Exit(1)
	}

	if err = (&khulnasoftscanner.KhulnasoftScannerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftScanner")
		os.Exit(1)
	}

	if err = (&khulnasoftcloudconnector.KhulnasoftCloudConnectorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftCloudConnector")
		os.Exit(1)
	}

	if err = (&khulnasoftlightning.KhulnasoftLightningReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Certs:  khulnasoftlightning.GetKECerts(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftKubeEnforcer")
		os.Exit(1)
	}

	if err = (&khulnasoftserver.KhulnasoftServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftServer")
		os.Exit(1)
	}

	if err = (&khulnasoftstarboard.KhulnasoftStarboardReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, controllerMessage, "controller", "KhulnasoftStarboard")
		os.Exit(1)
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
