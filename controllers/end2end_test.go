package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	khulnasoftv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/khulnasoft/v1alpha1"
	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	testingconsts "github.com/khulnasoft/khulnasoft-operator/test/consts"
	testutils "github.com/khulnasoft/khulnasoft-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	timeout             = time.Minute * 6
	interval            = time.Second * 30
	enforcerTimeout     = time.Minute * 3
	scannerTimeout      = time.Minute * 1
	KubeEnforcerTimeout = time.Minute * 5
	StarboardTimeout    = time.Minute * 2
)

var _ = Describe("Khulnasoft Controller", Serial, func() {
	localLog := logf.Log.WithName("KhulnasoftCspControllerTest")
	ubi := os.Getenv("RUN_UBI")

	Context("Initial deployment", func() {
		namespace := "khulnasoft"
		name := "khulnasoft"

		It("It should create KhulnasoftCsp Deployment", func() {
			instance := &operatorv1alpha1.KhulnasoftCsp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.KhulnasoftCspSpec{
					Infrastructure: &operatorv1alpha1.KhulnasoftInfrastructure{
						ServiceAccount: testingconsts.CspServiceAccount,
						Namespace:      testingconsts.NameSpace,
						Version:        testingconsts.Version,
						Requirements:   true,
					},
					Common: &operatorv1alpha1.KhulnasoftCommon{
						ImagePullSecret: testingconsts.ImagePullSecret,
						DbDiskSize:      testingconsts.DbDiskSize,
						DatabaseSecret: &operatorv1alpha1.KhulnasoftSecret{
							Name: testingconsts.DatabaseSecretName,
							Key:  testingconsts.DataBaseSecretKey,
						},
					},
					DbService: &operatorv1alpha1.KhulnasoftService{
						Replicas:    1,
						ServiceType: "ClusterIP",
						ImageData: &operatorv1alpha1.KhulnasoftImage{
							Registry:   testingconsts.Registry,
							Repository: testingconsts.DatabaseRepo,
							PullPolicy: "Always",
						},
					},
					GatewayService: &operatorv1alpha1.KhulnasoftService{
						Replicas:    1,
						ServiceType: "ClusterIP",
						ImageData: &operatorv1alpha1.KhulnasoftImage{
							Registry:   testingconsts.Registry,
							Repository: testingconsts.GatewayRepo,
							PullPolicy: "Always",
						},
					},
					ServerService: &operatorv1alpha1.KhulnasoftService{
						Replicas:    1,
						ServiceType: "LoadBalancer",
						ImageData: &operatorv1alpha1.KhulnasoftImage{
							Registry:   testingconsts.Registry,
							Repository: testingconsts.ServerRepo,
							PullPolicy: "Always",
						},
					},
					ServerEnvs: []corev1.EnvVar{
						{
							Name:  "LICENSE_TOKEN",
							Value: testutils.GetLicenseToken(),
						},
						{
							Name:  "ADMIN_PASSWORD",
							Value: testingconsts.ServerAdminPassword,
						},
						{
							Name:  "BATCH_INSTALL_NAME",
							Value: testingconsts.EnforcerGroupName,
						},
						{
							Name:  "BATCH_INSTALL_TOKEN",
							Value: testingconsts.EnforcerToken,
						},
						{
							Name:  "BATCH_INSTALL_GATEWAY",
							Value: fmt.Sprintf(testingconsts.GatewayServiceName, name),
						},
						{
							Name:  "KHULNASOFT_KE_GROUP_NAME",
							Value: testingconsts.KUbeEnforcerGroupName,
						},
						{
							Name:  "KHULNASOFT_KE_GROUP_TOKEN",
							Value: testingconsts.KubeEnforcerToken,
						},
					},
					Route:        true,
					RunAsNonRoot: false,
				},
			}
			// Adding ubi tags:
			if ubi == "true" {
				instance.Spec.GatewayService.ImageData.Tag = testingconsts.UbiImageTag
				instance.Spec.ServerService.ImageData.Tag = testingconsts.UbiImageTag
			}
			Expect(k8sClient.Create(context.Background(), instance)).Should(Succeed())

			cspLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			csp := &operatorv1alpha1.KhulnasoftCsp{}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), cspLookupKey, csp)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				if csp.Status.State != operatorv1alpha1.KhulnasoftDeploymentStateRunning {
					localLog.Info(fmt.Sprintf("csp state: %s", csp.Status.State))
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("It Should create KhulnasoftEnforcer DaemonSet", func() {
			instance := &operatorv1alpha1.KhulnasoftEnforcer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.KhulnasoftEnforcerSpec{
					Infrastructure: &operatorv1alpha1.KhulnasoftInfrastructure{
						ServiceAccount: testingconsts.CspServiceAccount,
						Version:        testingconsts.Version,
					},
					Common: &operatorv1alpha1.KhulnasoftCommon{
						ImagePullSecret: testingconsts.ImagePullSecret,
					},

					EnforcerService: &operatorv1alpha1.KhulnasoftService{
						ImageData: &operatorv1alpha1.KhulnasoftImage{
							Repository: testingconsts.EnforcerRepo,
							Registry:   testingconsts.Registry,
							PullPolicy: "IfNotPresent",
						},
					},
					RunAsNonRoot: false,
					Gateway: &operatorv1alpha1.KhulnasoftGatewayInformation{
						Host: fmt.Sprintf("%s-gateway", name),
						Port: testingconsts.GatewayPort,
					},
					Token: testingconsts.EnforcerToken,
				},
			}
			// Adding ubi tags:
			if ubi == "true" {
				instance.Spec.EnforcerService.ImageData.Tag = testingconsts.UbiImageTag
			}
			Expect(k8sClient.Create(context.Background(), instance)).Should(Succeed())

			enforcerLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			enforcer := &operatorv1alpha1.KhulnasoftEnforcer{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), enforcerLookupKey, enforcer)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				if enforcer.Status.State != operatorv1alpha1.KhulnasoftDeploymentStateRunning {
					localLog.Info(fmt.Sprintf("enforcer state: %s", enforcer.Status.State))
					return false
				}
				return true
			}, enforcerTimeout, interval).Should(BeTrue())
		})

		It("It Should create KhulnasoftScanner Deployment", func() {
			instance := &operatorv1alpha1.KhulnasoftScanner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.KhulnasoftScannerSpec{
					Infrastructure: &operatorv1alpha1.KhulnasoftInfrastructure{
						ServiceAccount: testingconsts.CspServiceAccount,
						Version:        testingconsts.Version,
					},
					Common: &operatorv1alpha1.KhulnasoftCommon{
						ImagePullSecret: testingconsts.ImagePullSecret,
					},
					ScannerService: &operatorv1alpha1.KhulnasoftService{
						Replicas: 1,
						ImageData: &operatorv1alpha1.KhulnasoftImage{
							Repository: testingconsts.ScannerRepo,
							Registry:   testingconsts.Registry,
							PullPolicy: "IfNotPresent",
						},
					},
					RunAsNonRoot: false,
					Login: &operatorv1alpha1.KhulnasoftLogin{
						Username: testingconsts.ServerAdminUser,
						Password: testingconsts.ServerAdminPassword,
						Host:     testingconsts.ServerHost,
						Token:    testingconsts.ScannerToken,
					},
				},
			}
			// Adding ubi tags:
			if ubi == "true" {
				instance.Spec.ScannerService.ImageData.Tag = testingconsts.UbiImageTag
			}
			Expect(k8sClient.Create(context.Background(), instance)).Should(Succeed())

			scannerLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			scanner := &operatorv1alpha1.KhulnasoftScanner{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), scannerLookupKey, scanner)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				if scanner.Status.State != operatorv1alpha1.KhulnasoftDeploymentStateRunning {
					localLog.Info(fmt.Sprintf("scanner state: %s", scanner.Status.State))
					return false
				}
				return true
			}, scannerTimeout, interval).Should(BeTrue())
		})

		It("It should create KhulnasoftKubeEnforcer Deployment", func() {
			instance := &operatorv1alpha1.KhulnasoftKubeEnforcer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.KhulnasoftKubeEnforcerSpec{
					Infrastructure: &operatorv1alpha1.KhulnasoftInfrastructure{
						ServiceAccount: testingconsts.KubeEnforcerServiceAccount,
						Namespace:      testingconsts.NameSpace,
						Version:        testingconsts.Version,
					},
					Config: operatorv1alpha1.KhulnasoftKubeEnforcerConfig{
						GatewayAddress:  testingconsts.GatewayAddress,
						ClusterName:     testingconsts.ClusterName,
						ImagePullSecret: testingconsts.ImagePullSecret,
					},
					KubeEnforcerService: &operatorv1alpha1.KhulnasoftService{
						ServiceType: "ClusterIP",
						ImageData: &operatorv1alpha1.KhulnasoftImage{
							Registry:   testingconsts.Registry,
							Repository: testingconsts.KeEnforcerRepo,
							PullPolicy: "Always",
						},
					},
					Token: testingconsts.KubeEnforcerToken,
					DeployStarboard: &operatorv1alpha1.KhulnasoftStarboardDetails{
						Infrastructure: &operatorv1alpha1.KhulnasoftInfrastructure{
							ServiceAccount: testingconsts.StarboardServiceAccount,
						},
						Config: operatorv1alpha1.KhulnasoftStarboardConfig{
							ImagePullSecret: testingconsts.StarboardImagePullSecret,
						},
						StarboardService: &operatorv1alpha1.KhulnasoftService{
							Replicas: 1,
						},
					},
				},
			}
			// Adding ubi tags:
			if ubi == "true" {
				instance.Spec.KubeEnforcerService.ImageData.Tag = testingconsts.UbiImageTag
			}
			Expect(k8sClient.Create(context.Background(), instance)).Should(Succeed())

			keLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			ke := &operatorv1alpha1.KhulnasoftKubeEnforcer{}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), keLookupKey, ke)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				if ke.Status.State != operatorv1alpha1.KhulnasoftDeploymentStateRunning {
					localLog.Info(fmt.Sprintf("ke state: %s", ke.Status.State))
					return false
				}
				return true
			}, KubeEnforcerTimeout, interval).Should(BeTrue())

			starboard := &khulnasoftv1alpha1.KhulnasoftStarboard{}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), keLookupKey, starboard)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				if ke.Status.State != operatorv1alpha1.KhulnasoftDeploymentStateRunning {
					localLog.Info(fmt.Sprintf("starboard state: %s", starboard.Status.State))
					return false
				}
				return true
			}, StarboardTimeout, interval).Should(BeTrue())
		})

		// Delete

		It("It should delete KhulnasoftKubeEnforcer Deployment", func() {

			keLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			ke := &operatorv1alpha1.KhulnasoftKubeEnforcer{}

			err := k8sClient.Get(context.Background(), keLookupKey, ke)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), ke)).Should(Succeed())
			}

		})

		It("It should delete KhulnasoftScanner Deployment", func() {

			scannerLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			scanner := &operatorv1alpha1.KhulnasoftScanner{}

			err := k8sClient.Get(context.Background(), scannerLookupKey, scanner)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), scanner)).Should(Succeed())
			}
		})

		It("It should delete KhulnasoftEnforcer DaemonSet", func() {

			enforcerLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			enforcer := &operatorv1alpha1.KhulnasoftEnforcer{}

			err := k8sClient.Get(context.Background(), enforcerLookupKey, enforcer)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), enforcer)).Should(Succeed())
			}
		})

		It("It should delete KhulnasoftCsp Deployment", func() {

			cspLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			csp := &operatorv1alpha1.KhulnasoftCsp{}

			err := k8sClient.Get(context.Background(), cspLookupKey, csp)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), csp)).Should(Succeed())
			}

		})
	})
})
