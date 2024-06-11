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

package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	khulnasoftv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/khulnasoft/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/controllers/khulnasoft/khulnasoftstarboard"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftcsp"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftdatabase"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftenforcer"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftgateway"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftkubeenforcer"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftscanner"
	"github.com/khulnasoft/khulnasoft-operator/controllers/operator/khulnasoftserver"
	testingconsts "github.com/khulnasoft/khulnasoft-operator/test/consts"
	testutils "github.com/khulnasoft/khulnasoft-operator/test/utils"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	//clientcmd clientCmd
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	encoderConfig := uzap.NewProductionEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	logStdout := os.Getenv("LOG_STDOUT")
	if logStdout == "true" {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.WriteTo(os.Stdout), zap.UseDevMode(false), zap.Encoder(encoder), zap.StacktraceLevel(zapcore.ErrorLevel)))

	} else {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(false), zap.Encoder(encoder), zap.StacktraceLevel(zapcore.ErrorLevel)))
	}

	log := logf.Log.WithName("BeforeSuite")
	fmt.Fprintln(GinkgoWriter, "hello")

	By("bootstrapping test environment")
	createKind := os.Getenv("CREATE_KIND")
	if createKind == "true" {
		log.Info("Running with Kind")
		os.Setenv("USE_EXISTING_CLUSTER", "true")
		testutils.KindClusterOperations("create")
	}

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = khulnasoftv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	PrepareEnv()
	// Start controllers
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftcsp.KhulnasoftCspReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftdatabase.KhulnasoftDatabaseReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftenforcer.KhulnasoftEnforcerReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftgateway.KhulnasoftGatewayReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftkubeenforcer.KhulnasoftKubeEnforcerReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
		Certs:  khulnasoftkubeenforcer.GetKECerts(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftscanner.KhulnasoftScannerReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())
	err = (&khulnasoftserver.KhulnasoftServerReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&khulnasoftstarboard.KhulnasoftStarboardReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	CleanEnv()
	createKind := os.Getenv("CREATE_KIND")
	if createKind == "true" {
		testutils.KindClusterOperations("delete")
	}
})

func CleanEnv() {
	//delete storage
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateServiceAccount())).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateClusterRole())).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateClusterRoleBinding())).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateRole())).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateRoleBinding())).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateHostPathProvisionerDeployment())).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateStorageClass())).Should(Succeed())
	//delete namespaces
	Expect(k8sClient.Delete(context.TODO(), testutils.CreateNamespace(testingconsts.Namespace))).Should(Succeed())
	//Expect(k8sClient.Delete(context.TODO(), test_utils.CreateNamespace("local-storage"))).Should(Succeed())
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
}

func PrepareEnv() {
	//create namespace
	Expect(k8sClient.Create(context.TODO(), testutils.CreateNamespace(testingconsts.Namespace))).Should(Succeed())
	// create secrets
	Expect(k8sClient.Create(context.TODO(), testutils.CreateKhulnasoftDatabasePassword(testingconsts.Namespace))).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), testutils.CreatePullingSecret(testingconsts.Namespace))).Should(Succeed())

	//create storage class
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateNamespace("local-storage"))).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateServiceAccount())).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateClusterRole())).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateClusterRoleBinding())).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateRole())).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateRoleBinding())).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateHostPathProvisionerDeployment())).Should(Succeed())
	//Expect(k8sClient.Create(context.TODO(), testutils.CreateStorageClass())).Should(Succeed())
}
