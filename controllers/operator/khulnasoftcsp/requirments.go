package khulnasoftcsp

import (
	"context"
	"fmt"
	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/controllers/ocp"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/secrets"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/serviceaccounts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	routev1 "github.com/openshift/api/route/v1"
)

/*	----------------------------------------------------------------------------------------------------------------
							Requirments
	----------------------------------------------------------------------------------------------------------------
*/

func (r *KhulnasoftCspReconciler) CreateImagePullSecret(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("Csp Requirements Phase", "Create Image Pull Secret")
	reqLogger.Info("Start creating khulnasoft images pull secret")

	secretName := fmt.Sprintf(consts.PullImageSecretName, cr.Name)
	if cr.Spec.Common != nil && cr.Spec.Common.ImagePullSecret != "" {
		secretName = cr.Spec.Common.ImagePullSecret
	}
	// Define a new secret object
	secret := secrets.CreatePullImageSecret(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-requirments", cr.Name),
		secretName,
		*cr.Spec.RegistryData)

	// Set KhulnasoftCspKind instance as the owner and controller
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

func (r *KhulnasoftCspReconciler) CreateDbPasswordSecret(cr *operatorv1alpha1.KhulnasoftCsp, name, key, password string) (reconcile.Result, error) {
	reqLogger := log.WithValues("Csp Requirments Phase", "Create Db Password Secret")
	reqLogger.Info("Start creating khulnasoft db password secret")

	// Define a new secret object
	secret := secrets.CreateSecret(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-requirments", cr.Name),
		"Secret for khulnasoft database password",
		name,
		key,
		password)

	// Set KhulnasoftCspKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this secret already exists
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Db Password Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Secret already exists - don't requeue
	reqLogger.Info("Skip reconcile: Khulnasoft Db Password Secret Already Exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{Requeue: true}, nil
}

func (r *KhulnasoftCspReconciler) CreateKhulnasoftServiceAccount(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("Csp Requirments Phase", "Create Khulnasoft Service Account")
	reqLogger.Info("Start creating khulnasoft service account")

	if len(cr.Spec.Common.ImagePullSecret) > 0 {
		foundSecret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Spec.Common.ImagePullSecret, Namespace: cr.Namespace}, foundSecret)
		if err != nil && errors.IsNotFound(err) {
			cr.Spec.Common.ImagePullSecret = ""
		}
	}

	// Define a new service account object
	sa := serviceaccounts.CreateServiceAccount(cr.Name,
		cr.Namespace,
		fmt.Sprintf("%s-requirments", cr.Name),
		cr.Spec.Infrastructure.ServiceAccount,
		cr.Spec.Common.ImagePullSecret)

	// Set KhulnasoftCspKind instance as the owner and controller
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

func (r *KhulnasoftCspReconciler) CreateRoute(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
	reqLogger := log.WithValues("Csp Requirments Phase", "Create route")
	reqLogger.Info("Start creating openshift route")

	route := ocp.NewRoute(cr.Name, cr.Namespace, fmt.Sprintf("%s-server", cr.Name), 8080)

	// Set KhulnasoftCspKind instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, route, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this route already exists
	found := &routev1.Route{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a New Khulnasoft Image Pull Secret", "Secret.Namespace", route.Namespace, "Secret.Name", route.Name)
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
