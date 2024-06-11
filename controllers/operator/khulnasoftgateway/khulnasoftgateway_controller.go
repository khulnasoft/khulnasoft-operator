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

package khulnasoftgateway

import (
	"context"
	syserrors "errors"
	"fmt"
	common2 "github.com/khulnasoft/khulnasoft-operator/controllers/common"
	ocp "github.com/khulnasoft/khulnasoft-operator/controllers/ocp"
	consts "github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s"
	secrets2 "github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
)

var log = logf.Log.WithName("controller_khulnasoftgateway")

// KhulnasoftGatewayReconciler reconciles a KhulnasoftGateway object
type KhulnasoftGatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftgateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.khulnasoft.com,resources=khulnasoftgateways/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route,resources=routes,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KhulnasoftGateway object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KhulnasoftGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftGateway")

	// Fetch the KhulnasoftGateway instance
	instance := &operatorv1alpha1.KhulnasoftGateway{}
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

	instance = r.updateGatewayObject(instance)
	r.Client.Update(context.Background(), instance)

	rbacHelper := common2.NewKhulnasoftRbacHelper(
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
		r.Client.Status().Update(context.Background(), instance)
	}

	if instance.Spec.Common.SplitDB {
		if instance.Spec.ExternalDb != nil &&
			(instance.Spec.AuditDB == nil ||
				(instance.Spec.AuditDB != nil && instance.Spec.AuditDB.Data == nil)) {
			reqLogger.Error(syserrors.New(
				"When using split DB with External DB, you must define auditDB information"),
				"Missing audit database information definition")
		}

		instance.Spec.AuditDB = common2.UpdateKhulnasoftAuditDB(instance.Spec.AuditDB, instance.Name)
	}

	if instance.Spec.GatewayService != nil {
		reqLogger.Info("Start Setup Khulnasoft Gateway")
		_, err = r.InstallGatewayService(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		_, err = r.InstallGatewayDeployment(instance)
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
func (r *KhulnasoftGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		Named("khulnasoftgateway-controller").
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		For(&operatorv1alpha1.KhulnasoftGateway{})

	// Openshift Route
	isOpenshift, _ := ocp.VerifyRouteAPI()
	if isOpenshift {
		builder.Owns(&routev1.Route{})
	}

	return builder.Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft Gateway
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftGatewayReconciler) updateGatewayObject(cr *operatorv1alpha1.KhulnasoftGateway) *operatorv1alpha1.KhulnasoftGateway {
	cr.Spec.Infrastructure = common2.UpdateKhulnasoftInfrastructure(cr.Spec.Infrastructure, cr.Name, cr.Namespace)
	cr.Spec.Common = common2.UpdateKhulnasoftCommon(cr.Spec.Common, cr.Name, false, false)

	if secrets2.CheckIfSecretExists(r.Client, consts.MtlsKhulnasoftGatewaySecretName, cr.Namespace) {
		log.Info(fmt.Sprintf("%s secret found, enabling mtls", consts.MtlsKhulnasoftGatewaySecretName))
		cr.Spec.Mtls = true
	}

	return cr
}

func (r *KhulnasoftGatewayReconciler) InstallGatewayDeployment(cr *operatorv1alpha1.KhulnasoftGateway) (reconcile.Result, error) {
	reqLogger := log.WithValues("Gateway Deployment Phase", "Install Database Deployment")
	reqLogger.Info("Start installing khulnasoft gateway deployment")

	// Define a new deployment object
	gatewayHelper := newKhulnasoftGatewayHelper(cr)
	deployment := gatewayHelper.newDeployment(cr)

	// Set KhulnasoftGateway instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	// Check if this deployment already exists
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Gateway Deployment", "Dervice.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = patch.DefaultAnnotator.SetLastAppliedAnnotation(deployment)
		if err != nil {
			reqLogger.Error(err, "Unable to set default for k8s-objectmatcher", err)
		}
		err = r.Client.Create(context.TODO(), deployment)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	if found != nil {
		update, err := k8s.CheckForK8sObjectUpdate("KhulnasoftGateway deployment", found, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}
		if update {
			err = r.Client.Update(context.Background(), deployment)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft Gateway: Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
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
			reqLogger.Error(err, "Khulnasoft Gateway: Failed to list pods.", "KhulnasoftGateway.Namespace", cr.Namespace, "KhulnasoftDatabase.Name", cr.Name)
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}
		podNames := k8s.PodNames(podList.Items)

		// Update status.Nodes if needed
		if !reflect.DeepEqual(podNames, cr.Status.Nodes) {
			cr.Status.Nodes = podNames
			_ = r.Client.Status().Update(context.Background(), cr)
		}

		currentState := cr.Status.State
		if !k8s.IsDeploymentReady(found, int(cr.Spec.GatewayService.Replicas)) {
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
	reqLogger.Info("Skip reconcile: Khulnasoft Gateway Deployment Already Exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *KhulnasoftGatewayReconciler) InstallGatewayService(cr *operatorv1alpha1.KhulnasoftGateway) (reconcile.Result, error) {
	reqLogger := log.WithValues("Gateway Requirements Phase", "Install Gateway Service")
	reqLogger.Info("Start installing khulnasoft gateway service")

	// Define a new Service object
	gatewayHelper := newKhulnasoftGatewayHelper(cr)
	service := gatewayHelper.newService(cr)

	// Set KhulnasoftGateway instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	// Check if this service already exists
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Gateway Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
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
	reqLogger.Info("Skip reconcile: Khulnasoft Gateway Service Already Exists", "Service.Namespace", found.Namespace, "Service.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

func (r *KhulnasoftGatewayReconciler) CreateRoute(cr *operatorv1alpha1.KhulnasoftGateway) (reconcile.Result, error) {
	reqLogger := log.WithValues("Gateway Requirements Phase", "Create route")
	reqLogger.Info("Start creating openshift route")

	gatewayHelper := newKhulnasoftGatewayHelper(cr)
	route := gatewayHelper.newRoute(cr)

	// Set KhulnasoftCspKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, route, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this route already exists
	found := &routev1.Route{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Gateway Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
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
