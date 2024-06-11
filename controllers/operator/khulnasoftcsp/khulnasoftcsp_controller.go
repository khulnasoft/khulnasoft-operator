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

package khulnasoftcsp

import (
	"context"
	syserrors "errors"
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logf.Log.WithName("controller_khulnasoftcsp")

// KhulnasoftCspReconciler reconciles a KhulnasoftCsp object
type KhulnasoftCspReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftcsps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftcsps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftcsps/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftgateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftenforcers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftkubeenforcers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftdatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.khulnasoftsec.com,resources=khulnasoftservers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KhulnasoftCsp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *KhulnasoftCspReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KhulnasoftCsp")

	// Fetch the KhulnasoftCsp instance
	instance := &v1alpha1.KhulnasoftCsp{}
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

	instance = r.updateCspObject(instance)

	if instance.Spec.Infrastructure.Requirements {
		reqLogger.Info("Start Setup Requirement For Khulnasoft CSP...")

		if instance.Spec.RegistryData != nil {
			marketplace := extra.IsMarketPlace()
			if !marketplace {
				reqLogger.Info("Start Setup Khulnasoft Image Secret Secret")
				_, err = r.CreateImagePullSecret(instance)
				if err != nil {
					return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
				}
			} else {
				reqLogger.Info("[Marketplace Mode] skipping creating of image pull secret, using images from RedHat repository with digest")
			}
		}
	}

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

	dbstatus := true
	if instance.Spec.DbService != nil {
		reqLogger.Info("Start Setup Secret For Database Password")
		password := extra.CreateRundomPassword()
		_, err = r.CreateDbPasswordSecret(
			instance,
			fmt.Sprintf(consts.ScalockDbPasswordSecretName, instance.Name),
			consts.ScalockDbPasswordSecretKey,
			password)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		reqLogger.Info("CSP Deployment: Start Setup Internal Khulnasoft Database (Not Recommended For Production Usage)")
		_, err = r.InstallKhulnasoftDatabase(instance)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		dbstatus, _ = r.WaitForDatabase(instance)
	} else if instance.Spec.ExternalDb != nil {
		if len(instance.Spec.ExternalDb.Password) != 0 {
			_, err = r.CreateDbPasswordSecret(
				instance,
				fmt.Sprintf(consts.ScalockDbPasswordSecretName, instance.Name),
				consts.ScalockDbPasswordSecretKey,
				instance.Spec.ExternalDb.Password)
			if err != nil {
				return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
			}
		} else {
			if instance.Spec.Common.DatabaseSecret != nil {
				exists := secrets.CheckIfSecretExists(r.Client, instance.Spec.Common.DatabaseSecret.Name, instance.Namespace)
				if !exists {
					reqLogger.Error(syserrors.New("For using external db you must define password, or define the secret name and key in common section!"), "Missing external database password definition")
				}
			} else {
				reqLogger.Error(syserrors.New("For using external db you must define password, or define the secret name and key in common section!"), "Missing external database password definition")
			}
		}

		// if splitDB ->
		//if no  AuditDB struct -> error
		// init AuditDB struct
		// Check if AuditDBSecret exist
		// if not -> create AuditDB secret using given password
		if instance.Spec.Common.SplitDB {
			if instance.Spec.ExternalDb != nil &&
				(instance.Spec.AuditDB == nil ||
					(instance.Spec.AuditDB != nil && instance.Spec.AuditDB.Data == nil)) {
				reqLogger.Error(syserrors.New(
					"When using split DB with External DB, you must define auditDB information"),
					"Missing audit database information definition")
			}

			instance.Spec.AuditDB = common.UpdateKhulnasoftAuditDB(instance.Spec.AuditDB, instance.Name)
			exist := secrets.CheckIfSecretExists(r.Client, instance.Spec.AuditDB.AuditDBSecret.Name, instance.Namespace)
			if !exist {
				_, err = r.CreateDbPasswordSecret(instance,
					instance.Spec.AuditDB.AuditDBSecret.Name,
					instance.Spec.AuditDB.AuditDBSecret.Key,
					instance.Spec.AuditDB.Data.Password)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	if dbstatus {
		if instance.Spec.GatewayService == nil {
			reqLogger.Error(syserrors.New("Missing Khulnasoft Gateway Deployment Data!, Please fix and redeploy template!"), "Khulnasoft CSP Deployment Missing Gateway Deployment Data!")
		}

		if instance.Spec.ServerService == nil {
			reqLogger.Error(syserrors.New("Missing Khulnasoft Server Deployment Data!, Please fix and redeploy template!"), "Khulnasoft CSP Deployment Missing Server Deployment Data!")
		}
		if instance.Spec.DeployKubeEnforcer != nil {
			keConfigMapData := map[string]string{
				"BATCH_INSTALL_GATEWAY": fmt.Sprintf(consts.GatewayServiceName, instance.Name),
				"KHULNASOFT_KE_GROUP_NAME":    "operator-default-ke-group",
				"KHULNASOFT_KE_GROUP_TOKEN":   consts.DefaultKubeEnforcerToken,
			}
			if instance.Spec.ServerConfigMapData != nil {
				for k, v := range keConfigMapData {
					instance.Spec.ServerConfigMapData[k] = v
				}
			} else {
				instance.Spec.ServerConfigMapData = keConfigMapData
			}
		}

		_, err = r.InstallKhulnasoftServer(instance)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		_, err = r.InstallKhulnasoftGateway(instance)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		serverGatewayStatus := r.GetGatewayServerState(instance)

		currentStatus := instance.Status.State
		serverGatewayReady := reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStateRunning, serverGatewayStatus)

		if serverGatewayReady {
			if !reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerUpdatePendingApproval, currentStatus) &&
				!reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerUpdateInProgress, currentStatus) &&
				!reflect.DeepEqual(v1alpha1.KhulnasoftEnforcerWaiting, currentStatus) &&
				!reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStateRunning, currentStatus) {
				instance.Status.State = v1alpha1.KhulnasoftDeploymentStateRunning
				_ = r.Client.Status().Update(context.Background(), instance)
			}
		} else {
			if !reflect.DeepEqual(serverGatewayStatus, currentStatus) {
				instance.Status.State = serverGatewayStatus
				_ = r.Client.Status().Update(context.Background(), instance)
			}
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}
	} else {
		reqLogger.Info("CSP Deployment: Waiting internal for database to start")
		if !reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStateWaitingDB, instance.Status.State) {
			instance.Status.State = v1alpha1.KhulnasoftDeploymentStateWaitingDB
			_ = r.Client.Status().Update(context.Background(), instance)
		}
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	}

	waitForEnforcer := false
	if instance.Spec.Enforcer != nil {
		_, err = r.InstallKhulnasoftEnforcer(instance)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}
		waitForEnforcer = true
	}

	waitForKubeEnforcer := false
	if instance.Spec.DeployKubeEnforcer != nil {
		_, err = r.InstallKhulnasoftKubeEnforcer(instance)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}
		waitForKubeEnforcer = true
	}

	if !reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentUpdateInProgress, instance.Status.State) &&
		(waitForKubeEnforcer || waitForEnforcer) {
		crStatus := r.WaitForEnforcersReady(instance, waitForEnforcer, waitForKubeEnforcer)
		if !reflect.DeepEqual(instance.Status.State, crStatus) {
			instance.Status.State = crStatus
			_ = r.Client.Status().Update(context.Background(), instance)
		}
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KhulnasoftCspReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.KhulnasoftCsp{}).
		Named("khulnasoftcsp-controller").
		WithOptions(controller.Options{Reconciler: r}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&operatorv1alpha1.KhulnasoftDatabase{}).
		Owns(&operatorv1alpha1.KhulnasoftServer{}).
		Owns(&operatorv1alpha1.KhulnasoftGateway{}).
		Owns(&operatorv1alpha1.KhulnasoftEnforcer{}).
		Owns(&operatorv1alpha1.KhulnasoftKubeEnforcer{})

	//isOpenshift, _ := ocp.VerifyRouteAPI()
	//if isOpenshift {
	//	builder.Owns(&routev1.Route{})
	//}
	return builder.Complete(r)
}

/*	----------------------------------------------------------------------------------------------------------------
							Khulnasoft CSP
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftCspReconciler) updateCspObject(cr *v1alpha1.KhulnasoftCsp) *v1alpha1.KhulnasoftCsp {
	admin := false
	license := false

	if len(cr.Spec.AdminPassword) != 0 {
		admin = true
	}

	if len(cr.Spec.LicenseToken) != 0 {
		license = true
	}

	registry := consts.Registry
	if cr.Spec.RegistryData != nil {
		if len(cr.Spec.RegistryData.URL) > 0 {
			registry = cr.Spec.RegistryData.URL
		}
	}

	cr.Spec.Infrastructure = common.UpdateKhulnasoftInfrastructure(cr.Spec.Infrastructure, cr.Name, cr.Namespace)
	cr.Spec.Common = common.UpdateKhulnasoftCommon(cr.Spec.Common, cr.Name, admin, license)

	if cr.Spec.ServerService == nil {
		cr.Spec.ServerService = &v1alpha1.KhulnasoftService{
			Replicas:    1,
			ServiceType: "ClusterIP",
			ImageData: &v1alpha1.KhulnasoftImage{
				Registry: registry,
			},
		}
	}

	if cr.Spec.GatewayService == nil {
		cr.Spec.GatewayService = &v1alpha1.KhulnasoftService{
			Replicas:    1,
			ServiceType: "ClusterIP",
			ImageData: &v1alpha1.KhulnasoftImage{
				Registry: registry,
			},
		}
	}

	if cr.Spec.DbService == nil && cr.Spec.ExternalDb == nil {
		cr.Spec.DbService = &v1alpha1.KhulnasoftService{
			Replicas:    1,
			ServiceType: "ClusterIP",
			ImageData: &v1alpha1.KhulnasoftImage{
				Registry: registry,
			},
		}
	}

	if cr.Spec.Enforcer != nil {
		if len(cr.Spec.Enforcer.Name) == 0 {
			cr.Spec.Enforcer.Name = "operator-default"
		}

		if len(cr.Spec.Enforcer.Gateway) == 0 {
			cr.Spec.Enforcer.Gateway = fmt.Sprintf("%s-gateway", cr.Name)
		}
	}

	return cr
}

func (r *KhulnasoftCspReconciler) InstallKhulnasoftDatabase(cr *v1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftDatabase Phase", "Install Khulnasoft Database")
	reqLogger.Info("Start installing KhulnasoftDatabase")

	// Define a new KhulnasoftDatabase object
	cspHelper := newKhulnasoftCspHelper(cr)
	khulnasoftdb := cspHelper.newKhulnasoftDatabase(cr)

	// Set KhulnasoftCsp instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, khulnasoftdb, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftDatabase already exists
	found := &v1alpha1.KhulnasoftDatabase{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: khulnasoftdb.Name, Namespace: khulnasoftdb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Database", "KhulnasoftDatabase.Namespace", khulnasoftdb.Namespace, "KhulnasoftDatabase.Name", khulnasoftdb.Name)
		err = r.Client.Create(context.TODO(), khulnasoftdb)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	if found != nil {
		size := khulnasoftdb.Spec.DbService.Replicas
		if found.Spec.DbService.Replicas != size {
			found.Spec.DbService.Replicas = size
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update khulnasoft database replicas.", "KhulnasoftDatabase.Namespace", found.Namespace, "KhulnasoftDatabase.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}
	}

	// KhulnasoftDatabase already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Database Exists", "KhulnasoftDatabase.Namespace", found.Namespace, "KhulnasoftDatabase.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

func (r *KhulnasoftCspReconciler) InstallKhulnasoftGateway(cr *v1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftGateway Phase", "Install Khulnasoft Gateway")
	reqLogger.Info("Start installing KhulnasoftGateway")

	// Define a new KhulnasoftGateway object
	cspHelper := newKhulnasoftCspHelper(cr)
	khulnasoftgw := cspHelper.newKhulnasoftGateway(cr)

	// Set KhulnasoftCsp instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, khulnasoftgw, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftGateway already exists
	found := &v1alpha1.KhulnasoftGateway{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: khulnasoftgw.Name, Namespace: khulnasoftgw.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Gateway", "KhulnasoftGateway.Namespace", khulnasoftgw.Namespace, "KhulnasoftGateway.Name", khulnasoftgw.Name)
		err = r.Client.Create(context.TODO(), khulnasoftgw)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	if found != nil {
		size := khulnasoftgw.Spec.GatewayService.Replicas
		if found.Spec.GatewayService.Replicas != size {
			found.Spec.GatewayService.Replicas = size
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update khulnasoft gateway replicas.", "KhulnasoftServer.Namespace", found.Namespace, "KhulnasoftServer.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}

		update := !reflect.DeepEqual(khulnasoftgw.Spec, found.Spec)

		reqLogger.Info("Checking for KhulnasoftGateway Upgrade", "khulnasoftgw", khulnasoftgw.Spec, "found", found.Spec, "update bool", update)
		if update {
			found.Spec = *(khulnasoftgw.Spec.DeepCopy())
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update KhulnasoftGateway.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// KhulnasoftGateway already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Gateway Exists", "KhulnasoftGateway.Namespace", found.Namespace, "KhulnasoftGateway.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

func (r *KhulnasoftCspReconciler) InstallKhulnasoftServer(cr *v1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftServer Phase", "Install Khulnasoft Server")
	reqLogger.Info("Start installing KhulnasoftServer")

	// Define a new KhulnasoftServer object
	cspHelper := newKhulnasoftCspHelper(cr)
	khulnasoftsr := cspHelper.newKhulnasoftServer(cr)

	// Set KhulnasoftCsp instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, khulnasoftsr, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftServer already exists
	found := &v1alpha1.KhulnasoftServer{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: khulnasoftsr.Name, Namespace: khulnasoftsr.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft KhulnasoftServer", "KhulnasoftServer.Namespace", khulnasoftsr.Namespace, "KhulnasoftServer.Name", khulnasoftsr.Name)
		err = r.Client.Create(context.TODO(), khulnasoftsr)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	if found != nil {
		size := khulnasoftsr.Spec.ServerService.Replicas
		if found.Spec.ServerService.Replicas != size {
			found.Spec.ServerService.Replicas = size
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update khulnasoft server replicas.", "KhulnasoftServer.Namespace", found.Namespace, "KhulnasoftServer.Name", found.Name)
				return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}

		update := !reflect.DeepEqual(khulnasoftsr.Spec, found.Spec)

		reqLogger.Info("Checking for KhulnasoftServer Upgrade", "khulnasoftsr", khulnasoftsr.Spec, "found", found.Spec, "update bool", update)
		if update {
			found.Spec = *(khulnasoftsr.Spec.DeepCopy())
			err = r.Client.Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update KhulnasoftServer.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// KhulnasoftServer already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Server Exists", "KhulnasoftServer.Namespace", found.Namespace, "KhulnasoftServer.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}

/*func (r *ReconcileKhulnasoftCsp) InstallKhulnasoftScanner(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftScanner Phase", "Install Khulnasoft Scanner")
	reqLogger.Info("Start installing KhulnasoftScanner")

	// Define a new KhulnasoftScanner object
	cspHelper := newKhulnasoftCspHelper(cr)
	scanner := cspHelper.newKhulnasoftScanner(cr)

	// Set KhulnasoftCsp instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, scanner, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KhulnasoftScanner already exists
	found := &operatorv1alpha1.KhulnasoftScanner{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: scanner.Name, Namespace: scanner.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Scanner", "KhulnasoftScanner.Namespace", scanner.Namespace, "KhulnasoftScanner.Name", scanner.Name)
		err = r.Client.Create(context.TODO(), scanner)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	} else if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	if found != nil {
		size := scanner.Spec.ScannerService.Replicas
		if found.Spec.ScannerService.Replicas != size {
			found.Spec.ScannerService.Replicas = size
			err = r.Client.Status().Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP: Failed to update khulnasoft scanner replicas.", "KhulnasoftScanner.Namespace", found.Namespace, "KhulnasoftScanner.Name", found.Name)
				return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}
	}

	// KhulnasoftScanner already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Scanner Exists", "KhulnasoftScanner.Namespace", found.Namespace, "KhulnasoftScanner.Name", found.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}*/

func (r *KhulnasoftCspReconciler) InstallKhulnasoftEnforcer(cr *v1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftEnforcer Phase", "Install Khulnasoft Enforcer")
	reqLogger.Info("Start installing KhulnasoftEnforcer")

	// Define a new KhulnasoftEnforcer object
	cspHelper := newKhulnasoftCspHelper(cr)
	enforcer := cspHelper.newKhulnasoftEnforcer(cr)

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

func (r *KhulnasoftCspReconciler) InstallKhulnasoftKubeEnforcer(cr *v1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftKubeEnforcer Phase", "Install Khulnasoft Enforcer")
	reqLogger.Info("Start installing KhulnasoftKubeEnforcer")

	// Define a new KhulnasoftEnforcer object
	cspHelper := newKhulnasoftCspHelper(cr)
	enforcer := cspHelper.newKhulnasoftKubeEnforcer(cr)

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

/*	----------------------------------------------------------------------------------------------------------------
							Check Functions - Internal Only
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftCspReconciler) WaitForDatabase(cr *v1alpha1.KhulnasoftCsp) (bool, error) {
	reqLogger := log.WithValues("CSP - KhulnasoftDatabase Phase", "Wait For Database")
	reqLogger.Info("Start waiting to khulnasoft database")

	ready, err := r.GetPostgresReady(
		cr,
		fmt.Sprintf(consts.DbDeployName, cr.Name),
		int(cr.Spec.DbService.Replicas))
	if err != nil {
		return false, err
	}

	auditReady := true
	if cr.Spec.Common.SplitDB {
		auditReady, err = r.GetPostgresReady(
			cr,
			fmt.Sprintf(consts.AuditDbDeployName, cr.Name),
			1)
		if err != nil {
			return false, err
		}
	}
	return ready && auditReady, nil
}

func (r *KhulnasoftCspReconciler) GetPostgresReady(cr *v1alpha1.KhulnasoftCsp, dbDeployName string, replicas int) (bool, error) {
	resource := appsv1.Deployment{}

	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      dbDeployName,
	}

	err := r.Client.Get(context.TODO(), selector, &resource)
	if err != nil {
		return false, err
	}

	return int(resource.Status.ReadyReplicas) == replicas, nil
}

func (r *KhulnasoftCspReconciler) GetGatewayServerState(cr *v1alpha1.KhulnasoftCsp) v1alpha1.KhulnasoftDeploymentState {
	reqLogger := log.WithValues("CSP - KhulnasoftServer and KhulnasoftGateway Phase", "Wait For Khulnasoft Gateway and Server")
	reqLogger.Info("Start waiting to khulnasoft gateway and server")

	gatewayState := v1alpha1.KhulnasoftDeploymentStateRunning
	gatewayFound := &v1alpha1.KhulnasoftGateway{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, gatewayFound)
	if err != nil {
		reqLogger.Error(err, "Unable to Get KhulnasoftGateway Object", "err", err)
		gatewayState = v1alpha1.KhulnasoftDeploymentStatePending
	} else {
		gatewayState = gatewayFound.Status.State
	}

	serverState := v1alpha1.KhulnasoftDeploymentStateRunning
	serverFound := &v1alpha1.KhulnasoftServer{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, serverFound)
	if err != nil {
		reqLogger.Error(err, "Unable to Get KhulnasoftServer Object", "err", err)
		serverState = v1alpha1.KhulnasoftDeploymentStatePending
	} else {
		serverState = serverFound.Status.State
	}

	cspStatus := v1alpha1.KhulnasoftDeploymentStateRunning

	if reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStatePending, gatewayState) ||
		reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentStatePending, serverState) {
		cspStatus = v1alpha1.KhulnasoftDeploymentStateWaitingKhulnasoft
	} else if reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentUpdateInProgress, gatewayState) ||
		reflect.DeepEqual(v1alpha1.KhulnasoftDeploymentUpdateInProgress, serverState) {
		cspStatus = v1alpha1.KhulnasoftDeploymentUpdateInProgress
	}

	return cspStatus
}

func (r *KhulnasoftCspReconciler) WaitForEnforcersReady(cr *v1alpha1.KhulnasoftCsp, validateEnforcer, validateKubeEnforcer bool) v1alpha1.KhulnasoftDeploymentState {
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

/*func (r *ReconcileKhulnasoftCsp) WaitForServer(cr *operatorv1alpha1.KhulnasoftCsp) (bool, error) {
	reqLogger := log.WithValues("Csp Wait For Khulnasoft Server Phase", "Wait For Khulnasoft Server")
	reqLogger.Info("Start waiting to khulnasoft server")

	ready, err := r.GetServerReady(cr)
	if err != nil {
		return false, err
	}

	return ready, nil
}

func (r *ReconcileKhulnasoftCsp) GetServerReady(cr *operatorv1alpha1.KhulnasoftCsp) (bool, error) {
	resource := appsv1.Deployment{}

	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      fmt.Sprintf(consts.ServerDeployName, cr.Name),
	}

	err := r.Client.Get(context.TODO(), selector, &resource)
	if err != nil {
		return false, err
	}

	return int(resource.Status.ReadyReplicas) == int(cr.Spec.ServerService.Replicas), nil
}

func (r *ReconcileKhulnasoftCsp) ScaleScannerCLI(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("CSP - Scale", "Scale Khulnasoft Scanner CLI")
	reqLogger.Info("Start get scanner cli data")

	// TODO:
	result, err := common.GetPendingScanQueue("administrator", cr.Spec.AdminPassword, fmt.Sprintf(consts.ServerServiceName, cr.Name))
	if err != nil {
		reqLogger.Info("Waiting for khulnasoft server to be up...")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Count of pending scan queue", "Pending Scan Queue", result.Count)

	if result.Count == 0 {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
	}

	nodes := &corev1.NodeList{}
	count := int64(0)
	err = r.Client.List(context.TODO(), nodes, &client.ListOptions{})
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
	}

	for index := 0; index < len(nodes.Items); index++ {
		if val, ok := nodes.Items[index].Labels["kubernetes.io/role"]; ok {
			if val == "node" {
				count++
			}
		} else if val, ok := nodes.Items[index].Labels["node-role.kubernetes.io/compute"]; ok {
			if val == "true" {
				count++
			}
		}
	}

	reqLogger.Info("Khulnasoft CSP Scanner Scale:", "Kubernetes Nodes Count:", count)

	if count == 0 {
		count = 1
	}

	scanners := result.Count / cr.Spec.Scale.ImagesPerScanner
	extraScanners := result.Count % cr.Spec.Scale.ImagesPerScanner

	if scanners < cr.Spec.Scale.Min {
		scanners = cr.Spec.Scale.Min
	} else {
		if extraScanners > 0 {
			scanners = scanners + 1
		}

		if (cr.Spec.Scale.Max * count) < scanners {
			scanners = (cr.Spec.Scale.Max * count)
		}
	}

	reqLogger.Info("Khulnasoft CSP Scanner Scale:", "Final Scanners Count:", scanners)

	found := &operatorv1alpha1.KhulnasoftScanner{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, found)
	if found != nil {
		reqLogger.Info(string(found.Spec.ScannerService.Replicas))
		reqLogger.Info(string(scanners))

		if found.Spec.ScannerService.Replicas == scanners {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
		}

		if result.Count > 0 {
			found.Spec.ScannerService.Replicas = scanners
			err = r.Client.Status().Update(context.Background(), found)
			if err != nil {
				reqLogger.Error(err, "Khulnasoft CSP Scanner Scale: Failed to update Khulnasoft Scanner.", "KhulnasoftScanner.Namespace", found.Namespace, "KhulnasoftScanner.Name", found.Name)
				return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, err
			}
		}
	}

	return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(0)}, nil
}*/
