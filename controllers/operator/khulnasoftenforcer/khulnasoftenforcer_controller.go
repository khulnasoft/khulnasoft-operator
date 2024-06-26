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

package khulnasoftenforcer

import (
	"context"
	syserrors "errors"
	"fmt"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
)

var log = logf.Log.WithName("controller_khulnasoftenforcer")

// KhulnasoftEnforcerReconciler reconciles a KhulnasoftEnforcer object
type KhulnasoftEnforcerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftenforcers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftenforcers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftenforcers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KhulnasoftEnforcer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KhulnasoftEnforcerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftEnforcer")

	// Fetch the KhulnasoftEnforcer instance
	instance := &operatorv1alpha1.KhulnasoftEnforcer{}
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

	instance = r.updateEnforcerObject(instance)
	r.Client.Update(context.Background(), instance)

	rbacHelper := common.NewKhulnasoftRbacHelper(
		instance.Spec.Infrastructure,
		instance.Name,
		instance.Namespace,
		instance.Spec.Common,
		r.Client,
		r.Scheme,
		instance)

	err = rbacHelper.CreateRBAC()
	if err != nil {
		return reconcile.Result{}, err
	}

	currentStatus := instance.Status.State
	if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, currentStatus) &&
		!reflect.DeepEqual(operatorv1alpha1.KhulnasoftEnforcerUpdatePendingApproval, currentStatus) &&
		!reflect.DeepEqual(operatorv1alpha1.KhulnasoftEnforcerUpdateInProgress, currentStatus) {
		instance.Status.State = operatorv1alpha1.KhulnasoftDeploymentStatePending
		_ = r.Client.Status().Update(context.Background(), instance)
	}

	if instance.Spec.EnforcerService != nil {
		if len(instance.Spec.Token) != 0 {
			instance.Spec.Secret = &operatorv1alpha1.Khulnasoftret{
				Name: fmt.Sprintf(consts.EnforcerTokenSecretName, instance.Name),
				Key:  consts.EnforcerTokenSecretKey,
			}

			_, err = r.InstallEnforcerToken(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if instance.Spec.Secret == nil {
			reqLogger.Error(syserrors.New("You must specifie the enforcer token or the token secret name and key"), "Missing enforcer token")
		} else {
			exists := secrets.CheckIfSecretExists(r.Client, instance.Spec.Secret.Name, instance.Namespace)
			if !exists {
				reqLogger.Error(syserrors.New("You must specifie the enforcer token or the token secret name and key"), "Missing enforcer token")

			}
		}

		_, err = r.addEnforcerConfigMap(instance)

		if err != nil {
			return reconcile.Result{}, err
		}

		_, err = r.InstallEnforcerDaemonSet(instance)

		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftEnforcerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("khulnasoftenforcer-controller").
		WithOptions(controller.Options{Reconciler: r}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.DaemonSet{}).
		For(&operatorv1alpha1.KhulnasoftEnforcer{}).
		Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft Enforcer
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftEnforcerReconciler) updateEnforcerObject(cr *operatorv1alpha1.KhulnasoftEnforcer) *operatorv1alpha1.KhulnasoftEnforcer {
	version := cr.Spec.Infrastructure.Version
	if len(version) == 0 {
		version = consts.LatestVersion
	}

	if cr.Spec.EnforcerService == nil {
		cr.Spec.EnforcerService = &operatorv1alpha1.KhulnasoftService{
			ImageData: &operatorv1alpha1.KhulnasoftImage{
				Repository: "enforcer",
				Registry:   consts.Registry,
				Tag:        version,
				PullPolicy: consts.PullPolicy,
			},
		}
	}

	cr.Spec.Infrastructure = common.UpdateKhulnasoftInfrastructure(cr.Spec.Infrastructure, cr.Name, cr.Namespace)
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
		cr.Spec.Mtls = true
	}

	return cr
}

func (r *KhulnasoftEnforcerReconciler) InstallEnforcerDaemonSet(cr *operatorv1alpha1.KhulnasoftEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("Khulnasoft Enforcer DaemonSet Phase", "Install Khulnasoft Enforcer DaemonSet")
	reqLogger.Info("Start installing enforcer")

	// Define a new DaemonSet object
	enforcerHelper := newKhulnasoftEnforcerHelper(cr)
	ds := enforcerHelper.CreateDaemonSet(cr)

	// Set KhulnasoftEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, ds, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Enforcer", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
		err = patch.DefaultAnnotator.SetLastAppliedAnnotation(ds)
		if err != nil {
			reqLogger.Error(err, "Unable to set default for k8s-objectmatcher", err)
		}
		err = r.Client.Create(context.TODO(), ds)
		if err != nil {
			return reconcile.Result{}, err
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

		update, err := k8s.CheckForK8sObjectUpdate("KhulnasoftEnforcer daemonset", found, ds)
		if err != nil {
			return reconcile.Result{}, err
		}

		if update && updateEnforcerApproved {
			err = r.Client.Update(context.Background(), ds)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft Enforcer: Failed to update Daemonset.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		} else if update && !updateEnforcerApproved {
			cr.Status.State = operatorv1alpha1.KhulnasoftEnforcerUpdatePendingApproval
			_ = r.Client.Status().Update(context.Background(), cr)
		} else {
			currentState := cr.Status.State
			if found.Status.DesiredNumberScheduled != found.Status.NumberReady {
				if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftEnforcerUpdateInProgress, currentState) &&
					!reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStatePending, currentState) {
					cr.Status.State = operatorv1alpha1.KhulnasoftEnforcerUpdateInProgress
					_ = r.Client.Status().Update(context.Background(), cr)
				}
			} else if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, currentState) &&
				found.Status.NumberReady > 0 {
				cr.Status.State = operatorv1alpha1.KhulnasoftDeploymentStateRunning
				_ = r.Client.Status().Update(context.Background(), cr)
			}
		}
	}

	// DaemonSet already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Enforcer DaemonSet Already Exists", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftEnforcerReconciler) InstallEnforcerToken(cr *operatorv1alpha1.KhulnasoftEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("Enforcer Requirements Phase", "Create Khulnasoft Enforcer Token Secret")
	reqLogger.Info("Start creating enforcer token secret")

	// Define a new DaemonSet object
	enforcerHelper := newKhulnasoftEnforcerHelper(cr)
	token := enforcerHelper.CreateTokenSecret(cr)
	// Adding token to the hashed data, for restart pods if token is changed
	hash, err := extra.GenerateMD5ForSpec(token.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftEnforcer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, token, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Secret already exists
	found := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: token.Name, Namespace: token.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Enforcer Token Secret", "Secret.Namespace", token.Namespace, "Secret.Name", token.Name)
		err = r.Client.Create(context.TODO(), token)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if !equality.Semantic.DeepDerivative(token.Data, found.Data) {
		found = token
		log.Info("Khulnasoft Enforcer: Updating Enforcer Token Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update Enforcer Token Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	// Secret already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Enforcer Token Secret Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftEnforcerReconciler) addEnforcerConfigMap(cr *operatorv1alpha1.KhulnasoftEnforcer) (reconcile.Result, error) {
	reqLogger := log.WithValues("Enforcer Requirements Phase", "Create ConfigMap")
	reqLogger.Info("Start creating ConfigMap")
	//reqLogger.Info(fmt.Sprintf("cr object : %v", cr.ObjectMeta))

	// Define a new ClusterRoleBinding object
	enforcerHelper := newKhulnasoftEnforcerHelper(cr)

	configMap := enforcerHelper.CreateConfigMap(cr)
	// Adding configmap to the hashed data, for restart pods if token is changed
	hash, err := extra.GenerateMD5ForSpec(configMap.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftScanner instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Enforcer: Creating a New ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
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
		log.Info("Khulnasoft Enforcer: Updating ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
		err := r.Client.Update(context.TODO(), foundConfigMap)
		if err != nil {
			log.Error(err, "Failed to update ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	reqLogger.Info("Skip reconcile: Khulnasoft Enforcer ConfigMap Exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
	return reconcile.Result{Requeue: true}, nil
}
