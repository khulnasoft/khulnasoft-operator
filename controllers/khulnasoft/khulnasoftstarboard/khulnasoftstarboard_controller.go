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

package khulnasoftstarboard

import (
	"context"
	"fmt"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	common2 "github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/rbac"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	khulnasoftv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/khulnasoft/v1alpha1"
)

var log = logf.Log.WithName("controller_KhulnasoftStarboard")

// KhulnasoftStarboardReconciler reconciles a KhulnasoftStarboard object
type KhulnasoftStarboardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=khulnasoft.khulnasoft.com,resources=khulnasoftstarboards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=khulnasoft.khulnasoft.com,resources=khulnasoftstarboards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=khulnasoft.khulnasoft.com,resources=khulnasoftstarboards/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KhulnasoftStarboard object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KhulnasoftStarboardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "req.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftStarboard")

	// Fetch the KhulnasoftStarboard instance
	instance := &khulnasoftv1alpha1.KhulnasoftStarboard{}
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
	instance = r.updateStarboardObject(instance)
	r.Client.Update(context.Background(), instance)

	if !reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStateRunning, instance.Status.State) &&
		!reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentUpdateInProgress, instance.Status.State) {
		instance.Status.State = v1alpha1.KhulnasoftDeploymentStatePending
		_ = r.Client.Status().Update(context.Background(), instance)
	}

	_, err = r.addStarboardClusterRole(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.createKhulnasoftStarboardServiceAccount(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if strings.ToLower(instance.Spec.Infrastructure.Platform) == consts.OpenShiftPlatform &&
		rbac.CheckIfClusterRoleExists(r.Client, consts.ClusterReaderRole) &&
		!rbac.CheckIfClusterRoleBindingExists(r.Client, consts.KhulnasoftKubeEnforcerSAClusterReaderRoleBind) {
		_, err = r.CreateClusterReaderRoleBinding(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	instance.Spec.StarboardService = r.updateStarboardServerObject(instance.Spec.StarboardService, instance.Spec.ImageData)

	_, err = r.addStarboardClusterRoleBinding(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addStarboardConfigMap(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addStarboardSecret(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.addStarboardDeployment(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftStarboardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("KhulnasoftStarboard-controller").
		WithOptions(controller.Options{Reconciler: r}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.ConfigMap{}).
		For(&khulnasoftv1alpha1.KhulnasoftStarboard{}).
		Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft Starboard
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftStarboardReconciler) addStarboardDeployment(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard deployment phase", "Create Deployment")
	reqLogger.Info("Start creating deployment")
	reqLogger.Info("Khulnasoft Starboard", "cr.Spec.Infrastructure.Version", cr.Spec.Infrastructure.Version)
	pullPolicy, registry, repository, tag := extra.GetImageData("starboard-operator", cr.Spec.Infrastructure.Version, cr.Spec.StarboardService.ImageData, true)

	starboardHelper := newKhulnasoftStarboardHelper(cr)
	deployment := starboardHelper.CreateStarboardDeployment(cr,
		"starboard-operator",
		"starboard-operator",
		registry,
		tag,
		pullPolicy,
		repository)

	// Set KhulnasoftStarboard instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Starboard: Creating a New deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
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

		update, err := k8s.CheckForK8sObjectUpdate("KhulnasoftStarboard deployment", found, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}

		if update {
			err = r.Client.Update(context.Background(), deployment)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft Starboard: Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}

		podList := &corev1.PodList{}
		labelSelector := labels.SelectorFromSet(found.Labels)
		listOps := &client.ListOptions{
			Namespace:     deployment.Namespace,
			LabelSelector: labelSelector,
		}
		err = r.Client.List(context.TODO(), podList, listOps)
		if err != nil {
			reqLogger.Error(err, "Khulnasoft Starboard: Failed to list pods.", "KhulnasoftStarboard.Namespace", cr.Namespace, "KhulnasoftStarboard.Name", cr.Name)
			return reconcile.Result{}, err
		}
		podNames := k8s.PodNames(podList.Items)

		// Update status.Nodes if needed
		if !reflect.DeepEqual(podNames, cr.Status.Nodes) {
			cr.Status.Nodes = podNames
		}

		currentState := cr.Status.State
		if !k8s.IsDeploymentReady(found, int(cr.Spec.StarboardService.Replicas)) {
			if !reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentUpdateInProgress, currentState) &&
				!reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStatePending, currentState) {
				cr.Status.State = v1alpha1.KhulnasoftDeploymentUpdateInProgress
				_ = r.Client.Status().Update(context.Background(), cr)
			}
		} else if !reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStateRunning, currentState) {
			cr.Status.State = v1alpha1.KhulnasoftDeploymentStateRunning
			_ = r.Client.Status().Update(context.Background(), cr)
		}
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Starboard Deployment Exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftStarboardReconciler) updateStarboardServerObject(serviceObject *v1alpha1.KhulnasoftService, StarboardImageData *v1alpha1.KhulnasoftImage) *v1alpha1.KhulnasoftService {

	if serviceObject == nil {
		serviceObject = &v1alpha1.KhulnasoftService{
			ImageData:   StarboardImageData,
			ServiceType: string(corev1.ServiceTypeClusterIP),
		}
	} else {
		if serviceObject.ImageData == nil {
			serviceObject.ImageData = StarboardImageData
		}
		if len(serviceObject.ServiceType) == 0 {
			serviceObject.ServiceType = string(corev1.ServiceTypeClusterIP)
		}

	}

	return serviceObject
}

func (r *KhulnasoftStarboardReconciler) updateStarboardObject(cr *khulnasoftv1alpha1.KhulnasoftStarboard) *khulnasoftv1alpha1.KhulnasoftStarboard {

	cr.Spec.Infrastructure = common2.UpdateKhulnasoftInfrastructureFull(cr.Spec.Infrastructure, cr.Name, cr.Namespace, "starboard")
	return cr
}

func (r *KhulnasoftStarboardReconciler) addStarboardClusterRole(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create Khulnasoft Starboard Cluster Role")
	reqLogger.Info("Start creating starboard cluster role")

	starboardHelper := newKhulnasoftStarboardHelper(cr)
	crole := starboardHelper.CreateStarboardClusterRole(cr.Name, cr.Namespace)

	// Set KhulnasoftStarboard instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, crole, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRole already exists
	found := &rbacv1.ClusterRole{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crole.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Starboard: Creating a New ClusterRole", "ClusterRole.Namespace", crole.Namespace, "ClusterRole.Name", crole.Name)
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
		log.Info("Khulnasoft Starboard: Updating ClusterRole", "ClusterRole.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
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

func (r *KhulnasoftStarboardReconciler) createKhulnasoftStarboardServiceAccount(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create Khulnasoft Starboard Service Account")
	reqLogger.Info("Start creating khulnasoft starboard service account")

	// Define a new service account object
	starboardHelper := newKhulnasoftStarboardHelper(cr)
	sa := starboardHelper.CreateStarboardServiceAccount(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-requirments", cr.Name),
		cr.Spec.Infrastructure.ServiceAccount)

	// Set KhulnasoftStarboardKind instance as the owner and controller
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

func (r *KhulnasoftStarboardReconciler) addStarboardClusterRoleBinding(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create ClusterRoleBinding")
	reqLogger.Info("Start creating ClusterRole")

	// Define a new ClusterRoleBinding object
	starboardHelper := newKhulnasoftStarboardHelper(cr)
	crb := starboardHelper.CreateClusterRoleBinding(cr.Name,
		cr.Namespace,
		"starboard-operator",
		"ke-crb",
		cr.Spec.Infrastructure.ServiceAccount,
		"starboard-operator")

	// Set KhulnasoftStarboard instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, crb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRoleBinding already exists
	found := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crb.Name, Namespace: crb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Starboard: Creating a New ClusterRoleBinding", "ClusterRoleBinding.Namespace", crb.Namespace, "ClusterRoleBinding.Name", crb.Name)
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

func (r *KhulnasoftStarboardReconciler) addStarboardConfigMap(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create ConfigMap")
	reqLogger.Info("Start creating ConfigMap")
	//reqLogger.Info(fmt.Sprintf("cr object : %v", cr.ObjectMeta))

	// Define a new ClusterRoleBinding object
	starboardHelper := newKhulnasoftStarboardHelper(cr)
	configMaps := []*corev1.ConfigMap{
		starboardHelper.CreateStarboardConftestConfigMap(cr.Name,
			cr.Namespace,
			"starboard-policies-config",
			"starboard-policies-configmap",
			cr.Spec.KubeEnforcerVersion,
		),
		starboardHelper.CreateStarboardConfigMap(cr.Name,
			cr.Namespace,
			"starboard",
			"starboard",
		),
	}

	configMapsData := make(map[string]string)

	for _, configMap := range configMaps {
		for k, v := range configMap.Data {
			configMapsData[k] = v
		}
	}

	hash, err := extra.GenerateMD5ForSpec(configMapsData)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftStarboard instance as the owner and controller
	requeue := true
	for _, configMap := range configMaps {
		// Set KhulnasoftStarboard instance as the owner and controller
		if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}
		// Check if ConfigMap already exists
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Khulnasoft Starboard: Creating a New ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
			err = r.Client.Create(context.TODO(), configMap)

			if err != nil {
				reqLogger.Error(err, fmt.Sprintf("Failed to create configmap name: %s", configMap.Name))
				return reconcile.Result{Requeue: true}, nil
			}
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}

		// Check if the ConfigMap Data, matches the found Data
		if !equality.Semantic.DeepDerivative(configMap.Data, foundConfigMap.Data) {
			foundConfigMap = configMap
			log.Info("Khulnasoft Starboard: Updating ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
			err := r.Client.Update(context.TODO(), foundConfigMap)
			if err != nil {
				log.Error(err, "Khulnasoft Starboard: Failed to update ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}

		// MutatingWebhookConfiguration already exists - don't requeue
		reqLogger.Info("Skip reconcile: Khulnasoft Starboard ConfigMap Exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
	}
	return reconcile.Result{Requeue: requeue}, nil
}

func (r *KhulnasoftStarboardReconciler) addStarboardSecret(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create Token Secret")
	reqLogger.Info("Start creating token secret")

	starboardHelper := newKhulnasoftStarboardHelper(cr)
	starboardSecret := starboardHelper.CreateStarboardSecret(cr.Name,
		cr.Namespace,
		"khulnasoft-starboard-token",
		"ke-token-secret",
	)

	hash, err := extra.GenerateMD5ForSpec(starboardSecret)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftStarboard instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, starboardSecret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: starboardSecret.Name, Namespace: starboardSecret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Starboard: Creating a New token secret", "Secret.Namespace", starboardSecret.Namespace, "Secret.Name", starboardSecret.Name)
		err = r.Client.Create(context.TODO(), starboardSecret)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Starboard Token Secret Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftStarboardReconciler) CreateImagePullSecret(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create Image Pull Secret")
	reqLogger.Info("Start creating khulnasoft images pull secret")

	// Define a new secret object
	secret := secrets.CreatePullImageSecret(
		cr.Name,
		cr.Namespace,
		"ke-image-pull-secret",
		cr.Spec.Config.ImagePullSecret,
		*cr.Spec.RegistryData)

	// Set KhulnasoftStarboardKind instance as the owner and controller
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

func (r *KhulnasoftStarboardReconciler) CreateClusterReaderRoleBinding(cr *khulnasoftv1alpha1.KhulnasoftStarboard) (reconcile.Result, error) {
	reqLogger := log.WithValues("Starboard Requirements Phase", "Create Starboard ClusterReaderRoleBinding")
	reqLogger.Info("Start creating Starboard ClusterReaderRoleBinding")

	crb := rbac.CreateClusterRoleBinding(
		cr.Name,
		cr.Namespace,
		consts.KhulnasoftStarboardSAClusterReaderRoleBind,
		fmt.Sprintf("%s-starboard-cluster-reader", cr.Name),
		"Deploy Khulnasoft Starboard Cluster Reader Role Binding",
		"khulnasoft-starboard-sa",
		consts.ClusterReaderRole)

	// Set KhulnasoftStarboard instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, crb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ClusterRoleBinding already exists
	found := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crb.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Starboard: Creating a New Starboard ClusterReaderRoleBinding", "ClusterReaderRoleBinding.Namespace", crb.Namespace, "ClusterReaderRoleBinding.Name", crb.Name)
		err = r.Client.Create(context.TODO(), crb)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// ClusterRoleBinding already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Starboard ClusterReaderRoleBinding Exists", "ClusterRoleBinding.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}
