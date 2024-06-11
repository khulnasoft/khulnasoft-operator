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

package khulnasoftkubeenforcer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	syserrors "errors"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/khulnasoft/khulnasoft-operator/apis/khulnasoft/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/rbac"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
)

var log = logf.Log.WithName("controller_khulnasoftkubeenforcer")

// KhulnasoftKubeEnforcerReconciler reconciles a KhulnasoftKubeEnforcer object
type KhulnasoftKubeEnforcerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Certs  *KubeEnforcerCertificates
}

type KubeEnforcerCertificates struct {
	CAKey      []byte
	CACert     []byte
	ServerKey  []byte
	ServerCert []byte
}

//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftkubeenforcers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftkubeenforcers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftkubeenforcers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KhulnasoftKubeEnforcer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KhulnasoftKubeEnforcerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	req.NamespacedName.Namespace = extra.GetCurrentNameSpace()
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftKubeEnforcer")

	if r.Certs == nil {
		reqLogger.Error(syserrors.New("Unable to create KubeEnforcer Certificates"), "Unable to create KubeEnforcer Certificates")
		return reconcile.Result{}, nil
	}
	// Fetch the KhulnasoftKubeEnforcer instance
	instance := &operatorv1alpha1.KhulnasoftKubeEnforcer{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the Memcached instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMemcachedMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(instance, consts.KhulnasoftKubeEnforcerFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.KubeEnforcerFinalizer(instance); err != nil {
				return ctrl.Result{}, err
			}

			// Remove KubeEnforcerFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, consts.KhulnasoftKubeEnforcerFinalizer)
			err := r.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(instance, consts.KhulnasoftKubeEnforcerFinalizer) {
		controllerutil.AddFinalizer(instance, consts.KhulnasoftKubeEnforcerFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	instance = r.updateKubeEnforcerObject(instance)
	r.Client.Update(context.Background(), instance)

	currentStatus := instance.Status.State
	if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, currentStatus) &&
		!reflect.DeepEqual(operatorv1alpha1.KhulnasoftEnforcerUpdatePendingApproval, currentStatus) &&
		!reflect.DeepEqual(operatorv1alpha1.KhulnasoftEnforcerUpdateInProgress, currentStatus) {
		instance.Status.State = operatorv1alpha1.KhulnasoftDeploymentStatePending
		_ = r.Client.Status().Update(context.Background(), instance)
	}

	if instance.Spec.Config.ImagePullSecret == "" && !extra.IsMarketPlace() {
		instance.Spec.Config.ImagePullSecret = "khulnasoft-registry-secret"
	}

	if instance.Spec.RegistryData != nil {
		_, err = r.CreateImagePullSecret(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	instance.Spec.Infrastructure = common.UpdateKhulnasoftInfrastructure(instance.Spec.Infrastructure, consts.KhulnasoftKubeEnforcerClusterRoleBidingName, instance.Namespace)

	_, err = r.addKubeEnforcerClusterRole(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.createKhulnasoftServiceAccount(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	//if strings.ToLower(instance.Spec.Infrastructure.Platform) == consts.OpenShiftPlatform &&
	//	rbac.CheckIfClusterRoleExists(r.Client, consts.ClusterReaderRole) &&
	//	!rbac.CheckIfClusterRoleBindingExists(r.Client, consts.KhulnasoftKubeEnforcerSAClusterReaderRoleBind) {
	//	_, err = r.CreateClusterReaderRoleBinding(instance)
	//	if err != nil {
	//		return reconcile.Result{}, err
	//	}
	//}

	instance.Spec.KubeEnforcerService = r.updateKubeEnforcerServerObject(instance.Spec.KubeEnforcerService, instance.Spec.ImageData)

	_, err = r.addKEClusterRoleBinding(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKubeEnforcerRole(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKERoleBinding(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKEValidatingWebhook(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKEMutatingWebhook(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKEConfigMap(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKESecretToken(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKESecretSSL(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKEService(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addKEDeployment(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.DeployStarboard != nil {
		r.installKhulnasoftStarboard(instance)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftKubeEnforcerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("khulnasoftkubeenforcer-controller").
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&admissionv1.ValidatingWebhookConfiguration{}).
		Owns(&admissionv1.MutatingWebhookConfiguration{}).
		Owns(&corev1.ConfigMap{}).
		For(&operatorv1alpha1.KhulnasoftKubeEnforcer{}).
		Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft Kube-Enforcer-Internal
	----------------------------------------------------------------------------------------------------------------
*/

func GetKECerts() *KubeEnforcerCertificates {
	certs, err := createKECerts()
	if err != nil {
		return nil
	}

	return certs
}

func createKECerts() (*KubeEnforcerCertificates, error) {
	certs := &KubeEnforcerCertificates{}
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(2020),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: false,
		Subject: pkix.Name{
			CommonName: "admission_ca",
		},
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return certs, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return certs, err
	}

	// caPEM is ca.crt
	// caPrivKeyPEM is ca.key

	// pem encode
	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return certs, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return certs, err
	}

	namespace := extra.GetCurrentNameSpace()

	// set up our server certificate
	cert := &x509.Certificate{
		BasicConstraintsValid: false,
		SerialNumber:          big.NewInt(2020),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		DNSNames:              []string{fmt.Sprintf("khulnasoft-kube-enforcer.%s.svc", namespace), fmt.Sprintf("khulnasoft-kube-enforcer.%s.svc.cluster.local", namespace)},
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("khulnasoft-kube-enforcer.%s.svc", namespace),
		},
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return certs, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return certs, err
	}

	// certPEM is server.crt
	// certPrivKeyPEM is server.key

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return certs, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return certs, err
	}

	certs = &KubeEnforcerCertificates{
		CAKey:      caPrivKeyPEM.Bytes(),
		CACert:     certPEM.Bytes(),
		ServerKey:  certPrivKeyPEM.Bytes(),
		ServerCert: certPEM.Bytes(),
	}
	return certs, nil
}

/*
----------------------------------------------------------------------------------------------------------------

	Khulnasoft Kube-Enforcer

----------------------------------------------------------------------------------------------------------------
*/
func (r *KhulnasoftKubeEnforcerReconciler) updateKubeEnforcerServerObject(serviceObject *operatorv1alpha1.KhulnasoftService, kubeEnforcerImageData *operatorv1alpha1.KhulnasoftImage) *operatorv1alpha1.KhulnasoftService {

	if serviceObject == nil {
		serviceObject = &operatorv1alpha1.KhulnasoftService{
			ImageData:   kubeEnforcerImageData,
			ServiceType: string(corev1.ServiceTypeClusterIP),
		}
	} else {
		if serviceObject.ImageData == nil {
			serviceObject.ImageData = kubeEnforcerImageData
		}
		if len(serviceObject.ServiceType) == 0 {
			serviceObject.ServiceType = string(corev1.ServiceTypeClusterIP)
		}

	}

	return serviceObject
}

func (r *KhulnasoftKubeEnforcerReconciler) updateKubeEnforcerObject(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) *operatorv1alpha1.KhulnasoftKubeEnforcer {
	if secrets.CheckIfSecretExists(r.Client, consts.MtlsKhulnasoftKubeEnforcerSecretName, cr.Namespace) {
		log.Info(fmt.Sprintf("%s secret found, enabling mtls", consts.MtlsKhulnasoftKubeEnforcerSecretName))
		cr.Spec.Mtls = true
	}
	return cr
}

func (r *KhulnasoftKubeEnforcerReconciler) addKEDeployment(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Deployment Phase", "Create Deployment")
	reqLogger.Info("Start creating deployment")

	pullPolicy, registry, repository, tag := extra.GetImageData("kube-enforcer", cr.Spec.Infrastructure.Version, cr.Spec.KubeEnforcerService.ImageData, cr.Spec.AllowAnyVersion)

	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	deployment := enforcerHelper.CreateKEDeployment(cr,
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName,
		"ke-deployment",
		registry,
		tag,
		pullPolicy,
		repository)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = patch.DefaultAnnotator.SetLastAppliedAnnotation(deployment)
		if err != nil {
			reqLogger.Error(err, "Unable to set default for k8s-objectmatcher", err)
		}

		err = r.Client.Create(context.TODO(), deployment)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if found != nil {

		updateEnforcerApproved := true
		if cr.Spec.EnforcerUpdateApproved != nil {
			updateEnforcerApproved = *cr.Spec.EnforcerUpdateApproved
		}

		update, err := k8s.CheckForK8sObjectUpdate("KhulnasoftKubeEnforcer deployment", found, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}

		if update && updateEnforcerApproved {
			err = r.Client.Update(context.Background(), deployment)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft KubeEnforcer: Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		} else if update && !updateEnforcerApproved {
			cr.Status.State = operatorv1alpha1.KhulnasoftEnforcerUpdatePendingApproval
			_ = r.Client.Status().Update(context.Background(), cr)
		} else {
			currentState := cr.Status.State
			if !k8s.IsDeploymentReady(found, 1) {
				if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftEnforcerUpdateInProgress, currentState) &&
					!reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStatePending, currentState) {
					cr.Status.State = operatorv1alpha1.KhulnasoftEnforcerUpdateInProgress
					_ = r.Client.Status().Update(context.Background(), cr)
				}
			} else if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, currentState) {
				cr.Status.State = operatorv1alpha1.KhulnasoftDeploymentStateRunning
				_ = r.Client.Status().Update(context.Background(), cr)
			}
		}
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer Deployment Exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKubeEnforcerClusterRole(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create Khulnasoft KubeEnforcer Cluster Role")
	reqLogger.Info("Start creating kube-enforcer cluster role")

	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	crole := enforcerHelper.CreateKubeEnforcerClusterRole(cr.Name, cr.Namespace)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, crole, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRole already exists
	found := &rbacv1.ClusterRole{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crole.Name}, found)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ClusterRole", "ClusterRole.Namespace", crole.Namespace, "ClusterRole.Name", crole.Name)
		err = r.Client.Create(context.TODO(), crole)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Check if the ClusterRole Rules, matches the found Rules
	equal, err := k8s.CompareByHash(crole.Rules, found.Rules)

	if err != nil {
		return reconcile.Result{}, err
	}

	if !equal {
		found = crole
		log.Info("Khulnasoft KubeEnforcer: Updating ClusterRole", "ClusterRole.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update ClusterRole", "ClusterRole.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	// ClusterRole already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft ClusterRole Exists", "ClusterRole.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKEClusterRoleBinding(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create ClusterRoleBinding")
	reqLogger.Info("Start creating ClusterRole")

	// Define a new ClusterRoleBinding object
	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	crb := enforcerHelper.CreateClusterRoleBinding(cr.Name,
		cr.Namespace,
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName,
		"ke-crb",
		cr.Spec.Infrastructure.ServiceAccount,
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, crb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRoleBinding already exists
	found := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crb.Name, Namespace: crb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ClusterRoleBinding", "ClusterRoleBinding.Namespace", crb.Namespace, "ClusterRoleBinding.Name", crb.Name)
		err = r.Client.Create(context.TODO(), crb)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// ClusterRoleBinding already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft ClusterRoleBinding Exists", "ClusterRoleBinding.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) CreateClusterReaderRoleBinding(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create KubeEnforcer ClusterReaderRoleBinding")
	reqLogger.Info("Start creating KubeEnforcer ClusterReaderRoleBinding")

	crb := rbac.CreateClusterRoleBinding(
		cr.Name,
		cr.Namespace,
		consts.KhulnasoftKubeEnforcerSAClusterReaderRoleBind,
		fmt.Sprintf("%s-kube-enforcer-cluster-reader", cr.Name),
		"Deploy Khulnasoft KubeEnforcer Cluster Reader Role Binding",
		"khulnasoft-kube-enforcer-sa",
		consts.ClusterReaderRole)

	// Set KhulnasoftKube-enforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, crb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRoleBinding already exists
	found := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crb.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New KubeEnfocer ClusterReaderRoleBinding", "ClusterReaderRoleBinding.Namespace", crb.Namespace, "ClusterReaderRoleBinding.Name", crb.Name)
		err = r.Client.Create(context.TODO(), crb)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// ClusterRoleBinding already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer ClusterReaderRoleBinding Exists", "ClusterRoleBinding.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) createKhulnasoftServiceAccount(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create Khulnasoft KubeEnforcer Service Account")
	reqLogger.Info("Start creating khulnasoft kube-enforcer service account")

	// Define a new service account object
	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	sa := enforcerHelper.CreateKEServiceAccount(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-requirments", cr.Name),
		cr.Spec.Infrastructure.ServiceAccount)

	// Set KhulnasoftKubeEnforcerKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, sa, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this service account already exists
	found := &corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: sa.Name, Namespace: sa.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Service Account", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
		err = r.Client.Create(context.TODO(), sa)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Service account already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Service Account Already Exists", "ServiceAccount.Namespace", found.Namespace, "ServiceAccount.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKubeEnforcerRole(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create Khulnasoft KubeEnforcer Role")
	reqLogger.Info("Start creating kube-enforcer role")

	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	role := enforcerHelper.CreateKubeEnforcerRole(cr.Name, cr.Namespace, consts.KhulnasoftKubeEnforcerClusterRoleBidingName, fmt.Sprintf("%s-requirments", cr.Name))

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, role, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Role already exists
	found := &rbacv1.Role{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ClusterRole", "ClusterRole.Namespace", role.Namespace, "ClusterRole.Name", role.Name)
		err = r.Client.Create(context.TODO(), role)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Check if the Role Rules, matches the found Rules
	equal, err := k8s.CompareByHash(role.Rules, found.Rules)

	if err != nil {
		return reconcile.Result{}, err
	}

	if !equal {
		found = role
		log.Info("Khulnasoft KubeEnforcer: Updating Role", "Role.Namespace", found.Namespace, "Role.Name", found.Name)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update Role", "Role.Namespace", found.Namespace, "Role.Name", found.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	// ClusterRole already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft ClusterRole Exists", "ClusterRole.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKERoleBinding(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create RoleBinding")
	reqLogger.Info("Start creating RoleBinding")

	// Define a new ClusterRoleBinding object
	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	rb := enforcerHelper.CreateRoleBinding(cr.Name,
		cr.Namespace,
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName,
		"ke-rb",
		cr.Spec.Infrastructure.ServiceAccount,
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, rb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this RoleBinding already exists
	found := &rbacv1.RoleBinding{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: rb.Name, Namespace: rb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ClusterRoleBinding", "ClusterRoleBinding.Namespace", rb.Namespace, "ClusterRoleBinding.Name", rb.Name)
		err = r.Client.Create(context.TODO(), rb)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// RoleBinding already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft RoleBinding Exists", "RoleBinding.Namespace", found.Namespace, "Role.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKEValidatingWebhook(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create ValidatingWebhookConfiguration")
	reqLogger.Info("Start creating ValidatingWebhookConfiguration")

	// Define a new ClusterRoleBinding object
	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	validWebhook := enforcerHelper.CreateValidatingWebhook(cr.Name,
		cr.Namespace,
		consts.KhulnasoftKubeEnforcerValidatingWebhookConfigurationName,
		"ke-validatingwebhook",
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName,
		r.Certs.CACert)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, validWebhook, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ValidatingWebhookConfiguration already exists
	found := &admissionv1.ValidatingWebhookConfiguration{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: validWebhook.Name, Namespace: validWebhook.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ValidatingWebhookConfiguration", "ValidatingWebhook.Namespace", validWebhook.Namespace, "ClusterRoleBinding.Name", validWebhook.Name)
		err = r.Client.Create(context.TODO(), validWebhook)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// ValidatingWebhookConfiguration already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft ValidatingWebhookConfiguration Exists", "ValidatingWebhookConfiguration.Namespace", found.Namespace, "ValidatingWebhookConfiguration.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKEMutatingWebhook(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create MutatingWebhookConfiguration")
	reqLogger.Info("Start creating MutatingWebhookConfiguration")

	// Define a new ClusterRoleBinding object
	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	mutateWebhook := enforcerHelper.CreateMutatingWebhook(cr.Name,
		cr.Namespace,
		consts.KhulnasoftKubeEnforcerMutantingWebhookConfigurationName,
		"ke-mutatingwebhook",
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName,
		r.Certs.CACert)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, mutateWebhook, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRoleBinding already exists
	found := &admissionv1.MutatingWebhookConfiguration{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: mutateWebhook.Name, Namespace: mutateWebhook.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New MutatingWebhookConfiguration", "MutatingWebhook.Namespace", mutateWebhook.Namespace, "ClusterRoleBinding.Name", mutateWebhook.Name)
		err = r.Client.Create(context.TODO(), mutateWebhook)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// MutatingWebhookConfiguration already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft MutatingWebhookConfiguration Exists", "MutatingWebhookConfiguration.Namespace", found.Namespace, "MutatingWebhookConfiguration.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKEConfigMap(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create ConfigMap")
	reqLogger.Info("Start creating ConfigMap")
	//reqLogger.Info(fmt.Sprintf("cr object : %v", cr.ObjectMeta))

	// Define a new ClusterRoleBinding object
	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	deployStarboard := false
	if cr.Spec.DeployStarboard != nil {
		deployStarboard = true
	}
	configMap := enforcerHelper.CreateKEConfigMap(cr.Name,
		cr.Namespace,
		"khulnasoft-csp-kube-enforcer",
		"ke-configmap",
		cr.Spec.Config.GatewayAddress,
		cr.Spec.Config.KubeBenchImage,
		cr.Spec.Config.ClusterName,
		deployStarboard)
	// Adding configmap to the hashed data, for restart pods if token is changed
	hash, err := extra.GenerateMD5ForSpec(configMap.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		err = r.Client.Create(context.TODO(), configMap)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Check if the ConfigMap Data, matches the found Data
	if !equality.Semantic.DeepDerivative(configMap.Data, foundConfigMap.Data) {
		foundConfigMap = configMap
		log.Info("Khulnasoft KubeEnforcer: Updating ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
		err := r.Client.Update(context.TODO(), foundConfigMap)
		if err != nil {
			log.Error(err, "Failed to update ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	// MutatingWebhookConfiguration already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer ConfigMap Exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKESecretToken(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create Token Secret")
	reqLogger.Info("Start creating token secret")

	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	tokenSecret := enforcerHelper.CreateKETokenSecret(cr.Name,
		cr.Namespace,
		"khulnasoft-kube-enforcer-token",
		"ke-token-secret",
		cr.Spec.Token)
	// Adding secret to the hashed data, for restart pods if token is changed
	hash, err := extra.GenerateMD5ForSpec(tokenSecret.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, tokenSecret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: tokenSecret.Name, Namespace: tokenSecret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New token secret", "Secret.Namespace", tokenSecret.Namespace, "Secret.Name", tokenSecret.Name)
		err = r.Client.Create(context.TODO(), tokenSecret)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if !equality.Semantic.DeepDerivative(tokenSecret.Data, found.Data) {
		found = tokenSecret
		log.Info("Khulnasoft Enforcer: Updating KubeEnforcer Token Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update KubeEnforcer Token Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer Token Secret Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKESecretSSL(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create SSL Secret")
	reqLogger.Info("Start creating ssl secret")

	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	sslSecret := enforcerHelper.CreateKESSLSecret(cr.Name,
		cr.Namespace,
		"kube-enforcer-ssl",
		"ke-ssl-secret",
		r.Certs.ServerKey,
		r.Certs.ServerCert)

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, sslSecret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: sslSecret.Name, Namespace: sslSecret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New ssl secret", "Secret.Namespace", sslSecret.Namespace, "Secret.Name", sslSecret.Name)
		err = r.Client.Create(context.TODO(), sslSecret)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer SSL Secret Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) addKEService(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create Service")
	reqLogger.Info("Start creating service")

	enforcerHelper := newKhulnasoftKubeEnforcerHelper(cr)
	service := enforcerHelper.CreateKEService(cr.Name,
		cr.Namespace,
		consts.KhulnasoftKubeEnforcerClusterRoleBidingName,
		"ke-service")

	// Set KhulnasoftKubeEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Creating a New service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer Service Exists", "Service.Namespace", found.Namespace, "Service.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftKubeEnforcerReconciler) CreateImagePullSecret(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer Requirements Phase", "Create Image Pull Secret")
	reqLogger.Info("Start creating khulnasoft images pull secret")

	// Define a new secret object
	secret := secrets.CreatePullImageSecret(
		cr.Name,
		cr.Namespace,
		"ke-image-pull-secret",
		cr.Spec.Config.ImagePullSecret,
		*cr.Spec.RegistryData)

	// Set KhulnasoftKubeEnforcerKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this secret already exists
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Image Pull Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Secret already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Image Pull Secret Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

// Starboard functions

func (r *KhulnasoftKubeEnforcerReconciler) installKhulnasoftStarboard(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KubeEnforcer KhulnasoftStarboard Phase", "Install Khulnasoft Starboard")
	reqLogger.Info("Start installing KhulnasoftStarboard")

	// Define a new KhulnasoftServer object
	khulnasoftStarboardHelper := newKhulnasoftKubeEnforcerHelper(cr)

	khulnasoftsb := khulnasoftStarboardHelper.newStarboard(cr)

	// Set KhulnasoftKube-enforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, khulnasoftsb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftServer already exists
	found := &v1alpha1.KhulnasoftStarboard{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: khulnasoftsb.Name, Namespace: khulnasoftsb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft KhulnasoftStarboard", "KhulnasoftStarboard.Namespace", khulnasoftsb.Namespace, "KhulnasoftStarboard.Name", khulnasoftsb.Name)
		err = r.Client.Create(context.TODO(), khulnasoftsb)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	if found != nil {
		size := khulnasoftsb.Spec.StarboardService.Replicas
		if found.Spec.StarboardService.Replicas != size {
			found.Spec.StarboardService.Replicas = size
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft Kube-enforcer: Failed to update khulnasoft starboard replicas.", "KhulnasoftStarboard.Namespace", found.Namespace, "KhulnasoftStarboard.Name", found.Name)
				return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}

		update := !reflect.DeepEqual(khulnasoftsb.Spec, found.Spec)

		reqLogger.Info("Checking for KhulnasoftStarboard Upgrade", "khulnasoftsb", khulnasoftsb.Spec, "found", found.Spec, "update bool", update)
		if update {
			found.Spec = *(khulnasoftsb.Spec.DeepCopy())
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft Kube-enforcer: Failed to update KhulnasoftStarboard.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// KhulnasoftStarboard already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Starboard Exists", "KhulnasoftStarboard.Namespace", found.Namespace, "KhulnasoftStarboard.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

// finalizers

func (r *KhulnasoftKubeEnforcerReconciler) KubeEnforcerFinalizer(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) error {
	reqLogger := log.WithValues("KubeEnforcer Finalizer Phase", "Remove KE-Webhooks")

	// Check if this ValidatingWebhookConfiguration exists
	reqLogger.Info("Start removing ValidatingWebhookConfiguration")
	validatingWebhookConfiguration := &admissionv1.ValidatingWebhookConfiguration{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: consts.KhulnasoftKubeEnforcerValidatingWebhookConfigurationName, Namespace: cr.Namespace}, validatingWebhookConfiguration)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: ValidatingWebhookConfiguration is not found")
	} else if err != nil && !errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Getting ValidatingWebhookConfiguration status encountered error")
		return err
	} else {
		reqLogger.Info("Khulnasoft KubeEnforcer: ValidatingWebhookConfiguration is found, attempting to delete...")
		err = r.Client.Delete(context.TODO(), validatingWebhookConfiguration)
		if err != nil {
			reqLogger.Info("Khulnasoft KubeEnforcer: Failed to delete ValidatingWebhookConfiguration")
			return err
		}
		reqLogger.Info("Successfully removed ValidatingWebhookConfiguration")
	}

	// Check if this mutatingWebhookConfiguration exists
	reqLogger.Info("Start removing MutatingWebhookConfiguration")
	mutatingWebhookConfiguration := &admissionv1.MutatingWebhookConfiguration{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: consts.KhulnasoftKubeEnforcerMutantingWebhookConfigurationName, Namespace: cr.Namespace}, mutatingWebhookConfiguration)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: mutatingWebhookConfiguration is not found")
	} else if err != nil && !errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Getting mutatingWebhookConfiguration status encountered error")
		return err
	} else {
		reqLogger.Info("Khulnasoft KubeEnforcer: mutatingWebhookConfiguration is found, attempting to delete...")
		err = r.Client.Delete(context.TODO(), mutatingWebhookConfiguration)
		if err != nil {
			reqLogger.Info("Khulnasoft KubeEnforcer: Failed to delete mutatingWebhookConfiguration")
			return err
		}
		reqLogger.Info("Successfully removed mutatingWebhookConfiguration")
	}

	// Check if this ClusterRoleBinding exists
	reqLogger.Info("Start removing ClusterRoleBinding")
	cRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: consts.KhulnasoftKubeEnforcerClusterRoleBidingName, Namespace: cr.Namespace}, cRoleBinding)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: cRoleBinding is not found")
	} else if err != nil && !errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Getting cRoleBinding status encountered error")
		return err
	} else {
		reqLogger.Info("Khulnasoft KubeEnforcer: cRoleBinding is found, attempting to delete...")
		err = r.Client.Delete(context.TODO(), cRoleBinding)
		if err != nil {
			reqLogger.Info("Khulnasoft KubeEnforcer: Failed to delete cRoleBinding")
			return err
		}
		reqLogger.Info("Successfully removed cRoleBinding")
	}

	// Check if this ClusterReaderRoleBinding exists
	reqLogger.Info("Start removing cRoleReaderRoleBinding")
	cRoleReaderRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: consts.KhulnasoftKubeEnforcerSAClusterReaderRoleBind, Namespace: cr.Namespace}, cRoleReaderRoleBinding)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: cRoleReaderRoleBinding is not found")
	} else if err != nil && !errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Getting cRoleReaderRoleBinding status encountered error")
		return err
	} else {
		reqLogger.Info("Khulnasoft KubeEnforcer: cRoleReaderRoleBinding is found, attempting to delete...")
		err = r.Client.Delete(context.TODO(), cRoleBinding)
		if err != nil {
			reqLogger.Info("Khulnasoft KubeEnforcer: Failed to delete cRoleReaderRoleBinding")
			return err
		}
		reqLogger.Info("Successfully removed cRoleReaderRoleBinding")
	}

	// Check if this ClusterRole exists
	reqLogger.Info("Start removing cRole")
	cRole := &rbacv1.ClusterRole{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: consts.KhulnasoftKubeEnforcerClusterRoleName}, cRole)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: cRole is not found")
	} else if err != nil && !errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft KubeEnforcer: Getting cRole status encountered error")
		return err
	} else {
		reqLogger.Info("Khulnasoft KubeEnforcer: cRole is found, attempting to delete...")
		err = r.Client.Delete(context.TODO(), cRole)
		if err != nil {
			reqLogger.Info("Khulnasoft KubeEnforcer: Failed to delete cRole")
			return err
		}
		reqLogger.Info("Successfully removed cRole")
	}

	reqLogger.Info("Successfully Finalized")

	return nil
}
