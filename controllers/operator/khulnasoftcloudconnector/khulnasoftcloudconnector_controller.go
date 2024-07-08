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

package khulnasoftcloudconnector

import (
	"context"
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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
)

var log = logf.Log.WithName("controller_khulnasoftcloudconnector")

// KhulnasoftCloudConnectorReconciler reconciles a KhulnasoftCloudConnector object
type KhulnasoftCloudConnectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=cloudconnectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=cloudconnectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=cloudconnectors/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the CloudConnector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *KhulnasoftCloudConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftCloudConnector")

	// Fetch the KhulnasoftScanner instance
	instance := &operatorv1alpha1.KhulnasoftCloudConnector{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
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

	instance = r.updateCloudConnectorObject(instance)

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

	if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, instance.Status.State) &&
		!reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentUpdateInProgress, instance.Status.State) {
		instance.Status.State = operatorv1alpha1.KhulnasoftDeploymentStatePending
		_ = r.Client.Status().Update(context.Background(), instance)
	}

	if instance.Spec.CloudConnectorService != nil {
		_, err = r.InstallCloudConnectorDeployment(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftCloudConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("khulnasoftcloudconnector-controller").
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		For(&operatorv1alpha1.KhulnasoftCloudConnector{}).
		Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft CloudConnector
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftCloudConnectorReconciler) updateCloudConnectorObject(cr *operatorv1alpha1.KhulnasoftCloudConnector) *operatorv1alpha1.KhulnasoftCloudConnector {
	version := cr.Spec.Infrastructure.Version
	if len(version) == 0 {
		version = consts.LatestVersion
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

	return cr
}

func (r *KhulnasoftCloudConnectorReconciler) InstallCloudConnectorDeployment(cr *operatorv1alpha1.KhulnasoftCloudConnector) (reconcile.Result, error) {
	reqLogger := log.WithValues("CloudConnector Deployment Phase", "Install CloudConnector Deployment")
	reqLogger.Info("Start installing khulnasoft scanner cli deployment")

	// Define a new deployment object
	scannerHelper := newKhulnasoftCloudConnectorHelper(cr)
	r.addCloudConnectorSecret(cr)
	r.addCloudConnectorConfigMap(cr)

	deployment := scannerHelper.newDeployment(cr)

	// Set KhulnasoftCloudConnector instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this deployment already exists
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft CloudConnector Deployment", "Dervice.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = patch.DefaultAnnotator.SetLastAppliedAnnotation(deployment)
		if err != nil {
			reqLogger.Error(err, "Unable to set default for k8s-objectmatcher", err)
		}
		err = r.Client.Create(context.TODO(), deployment)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if found != nil {
		update, err := k8s.CheckForK8sObjectUpdate("KhulnasoftCloudConnector deployment", found, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}
		if update {
			err = r.Client.Update(context.Background(), deployment)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CloudConnector: Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
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
			reqLogger.Error(err, "Khulnasoft CloudConnector: Failed to list pods.", "KhulnasoftCloudConnector.Namespace", cr.Namespace, "KhulnasoftCloudConnector.Name", cr.Name)
			return reconcile.Result{}, err
		}
		podNames := k8s.PodNames(podList.Items)

		// Update status.Nodes if needed
		if !reflect.DeepEqual(podNames, cr.Status.Nodes) {
			cr.Status.Nodes = podNames
		}

		currentState := cr.Status.State
		if !k8s.IsDeploymentReady(found, int(cr.Spec.CloudConnectorService.Replicas)) {
			if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentUpdateInProgress, currentState) &&
				!reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStatePending, currentState) {
				cr.Status.State = operatorv1alpha1.KhulnasoftDeploymentUpdateInProgress
				_ = r.Client.Status().Update(context.Background(), cr)
			}
		} else if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, currentState) {
			cr.Status.State = operatorv1alpha1.KhulnasoftDeploymentStateRunning
			_ = r.Client.Status().Update(context.Background(), cr)
		}
	}

	// Deployment already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft CloudConnector Deployment Already Exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftCloudConnectorReconciler) addCloudConnectorSecret(cr *operatorv1alpha1.KhulnasoftCloudConnector) (reconcile.Result, error) {
	reqLogger := log.WithValues("CloudConnector Requirements Phase", "Create CloudConnector Secret")
	reqLogger.Info("Start creating CloudConnector secret")

	scannerHelper := newKhulnasoftCloudConnectorHelper(cr)
	scannerSecret := scannerHelper.CreateTokenSecret(cr)
	// Adding secret to the hashed data, for restart pods if token is changed
	hash, err := extra.GenerateMD5ForSpec(scannerSecret.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftCloudConnector instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, scannerSecret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this object already exists
	found := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: scannerSecret.Name, Namespace: scannerSecret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft CloudConnector: Creating a New scanner secret", "Secret.Namespace", scannerSecret.Namespace, "Secret.Name", scannerSecret.Name)
		err = r.Client.Create(context.TODO(), scannerSecret)
		if err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Check if the secret Data, matches the found Data
	if !equality.Semantic.DeepDerivative(scannerSecret.Data, found.Data) {
		found = scannerSecret
		log.Info("Khulnasoft CloudConnector: Updating CloudConnector Token Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// object already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft CloudConnector Secret Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftCloudConnectorReconciler) addCloudConnectorConfigMap(cr *operatorv1alpha1.KhulnasoftCloudConnector) (reconcile.Result, error) {
	reqLogger := log.WithValues("CloudConnector Requirements Phase", "Create ConfigMap")
	reqLogger.Info("Start creating ConfigMap")

	// Define a new ConfigMap object
	scannerHelper := newKhulnasoftCloudConnectorHelper(cr)

	configMap := scannerHelper.CreateConfigMap(cr)
	// Adding configmap to the hashed data, for restart pods if token is changed
	hash, err := extra.GenerateMD5ForSpec(configMap.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum += hash

	// Set KhulnasoftCloudConnector instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft CloudConnector: Creating a New ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
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
		log.Info("Khulnasoft CloudConnector: Updating ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
		err := r.Client.Update(context.TODO(), foundConfigMap)
		if err != nil {
			log.Error(err, "Failed to update ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	reqLogger.Info("Skip reconcile: Khulnasoft CloudConnector ConfigMap Exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
	return reconcile.Result{Requeue: true}, nil
}
