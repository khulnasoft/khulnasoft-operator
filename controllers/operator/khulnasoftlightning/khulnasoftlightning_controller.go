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

package khulnasoftlightning

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	stderrors "errors"
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	corev1 "k8s.io/api/core/v1"
	"math/big"
	ctrl "sigs.k8s.io/controller-runtime"
	//"github.com/khulnasoft/khulnasoft-operator/controllers/common"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	//ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const maxRetries = 3
const retryDelay = 1 * time.Second

var log = logf.Log.WithName("controller_khulnasoftlightning")

// KhulnasoftLightningReconciler reconciles a KhulnasoftKubeEnforcer object
type KhulnasoftLightningReconciler struct {
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

func (r *KhulnasoftLightningReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := r.reconcileOnce(ctx, req)
		if err == nil {
			return result, nil
		}

		if errors.IsConflict(err) {
			// Conflict error encountered, retry after delay
			time.Sleep(retryDelay)
			continue
		}

		return result, err
	}

	return reconcile.Result{}, stderrors.New("exhausted max retries")
}

func (r *KhulnasoftLightningReconciler) reconcileOnce(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftLightning")

	// Fetch the KhulnasoftCsp instance
	instance := &v1alpha1.KhulnasoftLightning{}
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

	instance = r.updateLightningObject(instance)

	_, err = r.InstallKhulnasoftEnforcer(instance)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	_, err = r.InstallKhulnasoftKubeEnforcer(instance)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	waitForEnforcer := true
	waitForKubeEnforcer := true

	if !reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentUpdateInProgress, instance.Status.State) &&
		(waitForKubeEnforcer || waitForEnforcer) {
		crStatus := r.WaitForEnforcersReady(instance, waitForEnforcer, waitForKubeEnforcer)
		if !reflect.DeepEqual(instance.Status.State, crStatus) {
			instance.Status.State = crStatus
			_ = r.Client.Status().Update(context.Background(), instance)
		}
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	}
	reqLogger.Info("Finished Reconciling KhulnasoftLightning")
	return ctrl.Result{}, nil
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft Lightning
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftLightningReconciler) updateLightningObject(cr *v1alpha1.KhulnasoftLightning) *v1alpha1.KhulnasoftLightning {
	version := cr.Spec.Enforcer.Infrastructure.Version
	if len(version) == 0 {
		version = consts.LatestVersion
	}

	if cr.Spec.Enforcer.EnforcerService == nil {
		cr.Spec.Enforcer.EnforcerService = &v1alpha1.KhulnasoftService{
			ImageData: &v1alpha1.KhulnasoftImage{
				Repository: "enforcer",
				Registry:   consts.Registry,
				Tag:        version,
				PullPolicy: consts.PullPolicy,
			},
		}
	}

	cr.Spec.Enforcer.Infrastructure = common.UpdateKhulnasoftInfrastructure(cr.Spec.Enforcer.Infrastructure, cr.Name, cr.Namespace)
	cr.Spec.Common = common.UpdateKhulnasoftCommon(cr.Spec.Common, cr.Name, false, false)

	if cr.Spec.Common != nil {
		if len(cr.Spec.Common.ImagePullSecret) != 0 {
			exist := secrets.CheckIfSecretExists(r.Client, cr.Spec.Common.ImagePullSecret, cr.Namespace)
			if !exist {
				cr.Spec.Common.ImagePullSecret = consts.EmptyString
			}
		}
	}

	if secrets.CheckIfSecretExists(r.Client, consts.MtlsKhulnasoftEnforcerSecretName, cr.Namespace) {
		log.Info(fmt.Sprintf("%s secret found, enabling mtls", consts.MtlsKhulnasoftEnforcerSecretName))
		cr.Spec.Enforcer.Mtls = true
	}
	if secrets.CheckIfSecretExists(r.Client, consts.MtlsKhulnasoftKubeEnforcerSecretName, cr.Namespace) {
		log.Info(fmt.Sprintf("%s secret found, enabling mtls", consts.MtlsKhulnasoftKubeEnforcerSecretName))
		cr.Spec.KubeEnforcer.Mtls = true
	}
	return cr
}

func (r *KhulnasoftLightningReconciler) InstallKhulnasoftKubeEnforcer(cr *v1alpha1.KhulnasoftLightning) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftKubeEnforcer Phase", "Install Khulnasoft Enforcer")
	reqLogger.Info("Start installing KhulnasoftKubeEnforcer")

	// Define a new KhulnasoftEnforcer object
	lightningHelper := newKhulnasoftLightningHelper(cr)
	enforcer := lightningHelper.newKhulnasoftKubeEnforcer(cr)

	// Set KhulnasoftCsp instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, enforcer, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftKubeEnforcer already exists
	found := &v1alpha1.KhulnasoftKubeEnforcer{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: enforcer.Name, Namespace: enforcer.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft KubeEnforcer", "KhulnasoftKubeEnforcer.Namespace", enforcer.Namespace, "KhulnasoftKubeEnforcer.Name", enforcer.Name)
		err = r.Client.Create(context.TODO(), enforcer)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}
	// KhulnasoftEnforcer already exists - don't requeue

	if found != nil {
		update := !reflect.DeepEqual(enforcer.Spec, found.Spec)

		reqLogger.Info("Checking for KhulnasoftKubeEnforcer Upgrade", "kube-enforcer", enforcer.Spec, "found", found.Spec, "update bool", update)
		if update {
			found.Spec = *(enforcer.Spec.DeepCopy())
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update KhulnasoftKubeEnforcer.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}

	}

	reqLogger.Info("Skip reconcile: Khulnasoft KubeEnforcer Exists", "KhulnasoftKubeEnforcer.Namespace", found.Namespace, "KhulnasoftKubeEnforcer.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

func (r *KhulnasoftLightningReconciler) InstallKhulnasoftEnforcer(cr *v1alpha1.KhulnasoftLightning) (reconcile.Result, error) {
	reqLogger := log.WithValues("Lightning - KhulnasoftEnforcer Phase", "Install Khulnasoft Enforcer")
	reqLogger.Info("Start installing KhulnasoftEnforcer")

	// Define a new KhulnasoftEnforcer object
	lightningHelper := newKhulnasoftLightningHelper(cr)
	enforcer := lightningHelper.newKhulnasoftEnforcer(cr)

	// Set KhulnasoftCsp instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, enforcer, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftEnforcer already exists
	found := &v1alpha1.KhulnasoftEnforcer{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: enforcer.Name, Namespace: enforcer.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Enforcer", "KhulnasoftEnforcer.Namespace", enforcer.Namespace, "KhulnasoftEnforcer.Name", enforcer.Name)
		err = r.Client.Create(context.TODO(), enforcer)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}
	// KhulnasoftEnforcer already exists - don't requeue

	if found != nil {
		update := !reflect.DeepEqual(enforcer.Spec, found.Spec)

		reqLogger.Info("Checking for KhulnasoftEnforcer Upgrade", "enforcer", enforcer.Spec, "found", found.Spec, "update bool", update)
		if update {
			found.Spec = *(enforcer.Spec.DeepCopy())
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update KhulnasoftEnforcer.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}
	}

	reqLogger.Info("Skip reconcile: Khulnasoft Enforcer Exists", "KhulnasoftEnforcer.Namespace", found.Namespace, "KhulnasoftEnforcer.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

func (r *KhulnasoftLightningReconciler) WaitForEnforcersReady(cr *v1alpha1.KhulnasoftLightning, validateEnforcer, validateKubeEnforcer bool) v1alpha1.KhulnasoftDeploymentState {
	reqLogger := log.WithValues("CSP - KhulnasoftEnforcers Phase", "Wait For Khulnasoft Enforcer and KubeEnforcer")
	reqLogger.Info("Start waiting to khulnasoft enforcer and kube-enforcer")

	enforcerStatus := v1alpha1.KhulnasoftDeploymentStateRunning
	if validateEnforcer {
		enforcerFound := &v1alpha1.KhulnasoftEnforcer{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, enforcerFound)
		if err != nil {
			reqLogger.Info("Unable to Get KhulnasoftEnforcer Object", "err", err)
			enforcerStatus = v1alpha1.KhulnasoftDeploymentStatePending
		} else {
			enforcerStatus = enforcerFound.Status.State
		}

	}

	kubeEnforcerStatus := v1alpha1.KhulnasoftDeploymentStateRunning
	if validateKubeEnforcer {
		kubeEnforcerFound := &v1alpha1.KhulnasoftKubeEnforcer{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, kubeEnforcerFound)
		if err != nil {
			reqLogger.Info("Unable to Get KhulnasoftKubeEnforcer Object", "err", err)
			kubeEnforcerStatus = v1alpha1.KhulnasoftDeploymentStatePending
		} else {
			kubeEnforcerStatus = kubeEnforcerFound.Status.State
		}

	}

	returnStatus := v1alpha1.KhulnasoftDeploymentStateRunning

	if reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStatePending, enforcerStatus) ||
		reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStatePending, kubeEnforcerStatus) {
		returnStatus = v1alpha1.KhulnasoftEnforcerWaiting
	} else if reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerUpdateInProgress, enforcerStatus) ||
		reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerUpdateInProgress, kubeEnforcerStatus) {
		returnStatus = v1alpha1.KhulnasoftEnforcerUpdateInProgress
	} else if reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerUpdatePendingApproval, enforcerStatus) ||
		reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerUpdatePendingApproval, kubeEnforcerStatus) {
		returnStatus = v1alpha1.KhulnasoftEnforcerUpdatePendingApproval
	}

	return returnStatus
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

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftLightningReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KhulnasoftLightning{}).
		Named("khulnasoftlightning-controller").
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&v1alpha1.KhulnasoftDatabase{}).
		Owns(&v1alpha1.KhulnasoftEnforcer{}).
		Owns(&v1alpha1.KhulnasoftKubeEnforcer{})

	return builder.Complete(r)
}
