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

package khulnasoftserver

import (
	"context"
	syserrors "errors"
	"fmt"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/controllers/ocp"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
)

var log = logf.Log.WithName("controller_khulnasoftserver")

// KhulnasoftServerReconciler reconciles a KhulnasoftServer object
type KhulnasoftServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftservers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route,resources=routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KhulnasoftServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KhulnasoftServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftServer")

	// Fetch the KhulnasoftServer instance
	instance := &operatorv1alpha1.KhulnasoftServer{}
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

	instance = r.updateServerObject(instance)
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

	if !reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentStateRunning, instance.Status.State) &&
		!reflect.DeepEqual(operatorv1alpha1.KhulnasoftDeploymentUpdateInProgress, instance.Status.State) {
		instance.Status.State = operatorv1alpha1.KhulnasoftDeploymentStatePending
		_ = r.Client.Status().Update(context.Background(), instance)
	}

	if instance.Spec.ServerService != nil {
		reqLogger.Info("Start Setup Khulnasoft Server")
		_, err = r.InstallServerService(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		if len(instance.Spec.AdminPassword) > 0 {
			reqLogger.Info("Start Creating Admin Password Secret")
			_, err = r.CreateAdminPasswordSecret(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else {
			if instance.Spec.Common.AdminPassword != nil {
				exists := secrets.CheckIfSecretExists(r.Client, instance.Spec.Common.AdminPassword.Name, instance.Namespace)
				if !exists {
					reqLogger.Error(syserrors.New("Admin password secret that mentioned in common section don't exists"), "Please create first or pass the password")
				}
			}
		}

		if len(instance.Spec.LicenseToken) > 0 {
			reqLogger.Info("Start Creating License Token Secret")
			_, err = r.CreateLicenseSecret(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else {
			if instance.Spec.Common.KhulnasoftLicense != nil {
				exists := secrets.CheckIfSecretExists(r.Client, instance.Spec.Common.KhulnasoftLicense.Name, instance.Namespace)
				if !exists {
					reqLogger.Error(syserrors.New("Khulnasoft license secret that mentioned in common section don't exists"), "Please create first or pass the license")
				}
			}
		}

		if instance.Spec.Enforcer != nil {
			reqLogger.Info("Start Setup Khulnasoft Enforcer Token Secret")
			_, err = r.CreateEnforcerToken(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		if instance.Spec.Common.SplitDB {
			if instance.Spec.ExternalDb != nil &&
				(instance.Spec.AuditDB == nil ||
					(instance.Spec.AuditDB != nil && instance.Spec.AuditDB.Data == nil)) {
				reqLogger.Error(syserrors.New(
					"When using split DB with External DB, you must define auditDB information"),
					"Missing audit database information definition")
			}

			instance.Spec.AuditDB = common.UpdateKhulnasoftAuditDB(instance.Spec.AuditDB, instance.Name)
		}

		reqLogger.Info("Start Creating Khulnasoft server ConfigMap")
		_, err = r.CreateServerConfigMap(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Start Creating Khulnasoft Server Deployment...")
		_, err = r.InstallServerDeployment(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		if strings.ToLower(instance.Spec.Infrastructure.Platform) == consts.OpenShiftPlatform && instance.Spec.Route {
			_, err = r.CreateRoute(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		Named("khulnasoftserver-controller").
		WithOptions(controller.Options{Reconciler: r}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		For(&operatorv1alpha1.KhulnasoftServer{})

	isOpenshift, _ := ocp.VerifyRouteAPI()
	if isOpenshift {
		builder.Owns(&routev1.Route{})
	}

	return builder.Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft Server
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftServerReconciler) updateServerObject(cr *operatorv1alpha1.KhulnasoftServer) *operatorv1alpha1.KhulnasoftServer {
	admin := false
	license := false

	if len(cr.Spec.AdminPassword) != 0 {
		admin = true
	}

	if len(cr.Spec.LicenseToken) != 0 {
		license = true
	}

	cr.Spec.Infrastructure = common.UpdateKhulnasoftInfrastructure(cr.Spec.Infrastructure, cr.Name, cr.Namespace)
	cr.Spec.Common = common.UpdateKhulnasoftCommon(cr.Spec.Common, cr.Name, admin, license)

	if cr.Spec.Enforcer != nil {
		if len(cr.Spec.Enforcer.Name) == 0 {
			cr.Spec.Enforcer.Name = "operator-default"
		}

		if len(cr.Spec.Enforcer.Gateway) == 0 {
			cr.Spec.Enforcer.Gateway = fmt.Sprintf("%s-gateway", cr.Name)
		}
	}

	if secrets.CheckIfSecretExists(r.Client, consts.MtlsKhulnasoftWebSecretName, cr.Namespace) {
		log.Info(fmt.Sprintf("%s secret found, enabling mtls", consts.MtlsKhulnasoftWebSecretName))
		cr.Spec.Mtls = true
	}

	return cr
}

func (r *KhulnasoftServerReconciler) InstallServerDeployment(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Deployment Phase", "Install Khulnasoft Server Deployment")
	reqLogger.Info("Start installing khulnasoft server deployment")

	// Define a new deployment object
	serverHelper := newKhulnasoftServerHelper(cr)
	deployment := serverHelper.newDeployment(cr)

	// Set KhulnasoftServer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this deployment already exists
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Server Deployment", "Dervice.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
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
		update, err := k8s.CheckForK8sObjectUpdate("KhulnasoftServer deployment", found, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}
		if update {
			err = r.Client.Update(context.Background(), deployment)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft Server: Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
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
			reqLogger.Error(err, "Khulnasoft Server: Failed to list pods.", "KhulnasoftServer.Namespace", cr.Namespace, "KhulnasoftServer.Name", cr.Name)
			return reconcile.Result{}, err
		}

		err = r.Client.List(context.TODO(), podList, listOps)
		if err != nil {
			reqLogger.Error(err, "KhulnasoftServer: Failed to list pods.", "KhulnasoftServer.Namespace", cr.Namespace, "KhulnasoftServer.Name", cr.Name)
			return reconcile.Result{}, err
		}

		podNames := k8s.PodNames(podList.Items)

		// Update status.Nodes if needed
		//if len(cr.Status.Nodes) == 0 {
		//	cr.Status.Nodes = podNames
		//}
		nodes := cr.Status.Nodes
		//var podsToAppend []string
		//for _, pod := range podNames {
		//	addPodName := true
		//	for _, node := range nodes {
		//		if pod == node {
		//			addPodName = false
		//		}
		//	}
		//	if addPodName {
		//		podsToAppend = append(podsToAppend, pod)
		//	}
		//}
		//if len(podsToAppend) > 0 {
		//	for _, pod := range podsToAppend {
		//		cr.Status.Nodes = append(cr.Status.Nodes, pod)
		//	}
		//	err := r.Client.Status().Update(context.Background(), cr)
		//	if err != nil {
		//		return reconcile.Result{}, err
		//	}
		//}

		// Update status.Nodes if needed
		if !reflect.DeepEqual(podNames, nodes) {
			cr.Status.Nodes = podNames
			r.Client.Status().Update(context.TODO(), cr)
		}

		currentState := cr.Status.State
		if !k8s.IsDeploymentReady(found, int(cr.Spec.ServerService.Replicas)) {
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
	reqLogger.Info("Skip reconcile: Khulnasoft Server Deployment Already Exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftServerReconciler) InstallServerService(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Requirements Phase", "Install Server Service")
	reqLogger.Info("Start installing khulnasoft server service")

	// Define a new Service object
	serverHelper := newKhulnasoftServerHelper(cr)
	service := serverHelper.newService(cr)

	// Set KhulnasoftServer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this service already exists
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Server Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if !reflect.DeepEqual(found.Spec.Type, service.Spec.Type) {
		service.Spec.ClusterIP = found.Spec.ClusterIP
		service.SetResourceVersion(found.GetResourceVersion())

		err = r.Client.Update(context.Background(), service)
		if err != nil {
			reqLogger.Error(err, "Khulnasoft Server: Failed to update Service.", "Service.Namespace", found.Namespace, "Service.Name", found.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Service already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Server Service Already Exists", "Service.Namespace", found.Namespace, "Service.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftServerReconciler) CreateAdminPasswordSecret(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Requirements Phase", "Create Server Secrets")
	reqLogger.Info("Start creating khulnasoft server admin password secret")

	// Define a new Secrets object
	secret := secrets.CreateSecret(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-server", cr.Name),
		"Secret for khulnasoft admin password",
		cr.Spec.Common.AdminPassword.Name,
		cr.Spec.Common.AdminPassword.Key,
		cr.Spec.AdminPassword)

	// Set KhulnasoftServer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this service already exists
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Server Admin Password Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Secrets already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Server Admin Password Secret Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftServerReconciler) CreateLicenseSecret(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Requirements Phase", "Create Server Secrets")
	reqLogger.Info("Start creating khulnasoft server license secret")

	// Define a new Secrets object
	secret := secrets.CreateSecret(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-server", cr.Name),
		"Secret for khulnasoft license token",
		cr.Spec.Common.KhulnasoftLicense.Name,
		cr.Spec.Common.KhulnasoftLicense.Key,
		cr.Spec.LicenseToken)

	// Set KhulnasoftServer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this service already exists
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Server License Token Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Secrets already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Server License Token Secret Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftServerReconciler) CreateServerConfigMap(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Requirements Phase", "Create Server ConfigMap")
	reqLogger.Info("Start creating khulnasoft server configMap")

	// Define a new ConfigMap object
	serverHelper := newKhulnasoftServerHelper(cr)

	configMap := serverHelper.CreateConfigMap(cr)
	hash, err := extra.GenerateMD5ForSpec(configMap.Data)
	if err != nil {
		return reconcile.Result{}, err
	}
	cr.Spec.ConfigMapChecksum = hash

	// Set KhulnasoftServer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Khulnasoft Server: Creating a New ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
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
		log.Info("Khulnasoft Server: Updating ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
		err := r.Client.Update(context.TODO(), foundConfigMap)
		if err != nil {
			log.Error(err, "Khulnasoft Server: Failed to update ConfigMap", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	reqLogger.Info("Skip reconcile: Khulnasoft Server ConfigMap Exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)

	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftServerReconciler) CreateRoute(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Requirements Phase", "Create route")
	reqLogger.Info("Start creating openshift route")

	serverHelper := newKhulnasoftServerHelper(cr)
	route := serverHelper.newRoute(cr)

	// Set KhulnasoftCspKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, route, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this route already exists
	found := &routev1.Route{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Server Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.Client.Create(context.TODO(), route)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Route already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Route Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

/*	----------------------------------------------------------------------------------------------------------------
							Enforcer
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftServerReconciler) CreateEnforcerToken(cr *operatorv1alpha1.KhulnasoftServer) (reconcile.Result, error) {
	reqLogger := log.WithValues("KhulnasoftServer Requirements Phase", "Create Enforcer Token")
	reqLogger.Info("Start creating khulnasoft default enforcer token secret")

	// Generate token
	token := extra.CreateRundomPassword()

	// Define a new secret object
	secret := secrets.CreateSecret(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-enforcer", cr.Name),
		"Enforcer token for default enforcer group",
		fmt.Sprintf("%s-enforcer-token", cr.Name),
		"token",
		token)

	// Set KhulnasoftCspKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this secret already exists
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Default Enforcer Token Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Secret already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Default Enforcer Token Secret Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}
