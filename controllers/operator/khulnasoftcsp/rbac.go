package khulnasoftcsp

/*	----------------------------------------------------------------------------------------------------------------
							RBAC
	----------------------------------------------------------------------------------------------------------------
*/

//func (r *ReconcileKhulnasoftCsp) CreateClusterRole(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
//	reqLogger := log.WithValues("CSP - RBAC Phase", "Create ClusterRole")
//	reqLogger.Info("Start creating ClusterRole")
//
//	// Define a new ClusterRole object
//	rbacHelper := common.NewKhulnasoftRbacHelper(cr.Spec.Infrastructure, cr.Name)
//	crole := rbacHelper.NewDiscoveryClusterRole(cr.Name, cr.Namespace)
//
//	// Set KhulnasoftCsp instance as the owner and controller
//	if err := controllerutil.SetControllerReference(cr, crole, r.scheme); err != nil {
//		return reconcile.Result{}, err
//	}
//
//	// Check if this ClusterRole already exists
//	found := &rbacv1.ClusterRole{}
//	err := r.client.Get(context.TODO(), types.NamespacedName{Name: crole.Name, Namespace: crole.Namespace}, found)
//	if err != nil && errors.IsNotFound(err) {
//		reqLogger.Info("Khulnasoft CSP: Creating a New ClusterRole", "ClusterRole.Namespace", crole.Namespace, "ClusterRole.Name", crole.Name)
//		err = r.client.Create(context.TODO(), crole)
//		if err != nil {
//			return reconcile.Result{Requeue: true}, nil
//		}
//
//		return reconcile.Result{}, nil
//	} else if err != nil {
//		return reconcile.Result{}, err
//	}
//
//	// ClusterRole already exists - don't requeue
//	reqLogger.Info("Skip reconcile: Khulnasoft ClusterRole Exists", "ClusterRole.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
//	return reconcile.Result{Requeue: true}, nil
//}
//
//func (r *ReconcileKhulnasoftCsp) CreateClusterRoleBinding(cr *operatorv1alpha1.KhulnasoftCsp) (reconcile.Result, error) {
//	reqLogger := log.WithValues("CSP - RBAC Phase", "Create ClusterRoleBinding")
//	reqLogger.Info("Start creating ClusterRole")
//
//	// Define a new ClusterRoleBinding object
//	rbacHelper := common.NewKhulnasoftRbacHelper(cr.Spec.Infrastructure, cr.Name)
//	crb := rbacHelper.NewDiscoveryClusterRoleBinding(cr.Name, cr.Namespace, cr.Spec.Infrastructure.ServiceAccount)
//
//	// Set KhulnasoftCsp instance as the owner and controller
//	if err := controllerutil.SetControllerReference(cr, crb, r.scheme); err != nil {
//		return reconcile.Result{}, err
//	}
//
//	// Check if this ClusterRoleBinding already exists
//	found := &rbacv1.ClusterRoleBinding{}
//	err := r.client.Get(context.TODO(), types.NamespacedName{Name: crb.Name, Namespace: crb.Namespace}, found)
//	if err != nil && errors.IsNotFound(err) {
//		reqLogger.Info("Khulnasoft CSP: Creating a New ClusterRoleBinding", "ClusterRoleBinding.Namespace", crb.Namespace, "ClusterRoleBinding.Name", crb.Name)
//		err = r.client.Create(context.TODO(), crb)
//		if err != nil {
//			return reconcile.Result{Requeue: true}, nil
//		}
//
//		return reconcile.Result{}, nil
//	} else if err != nil {
//		return reconcile.Result{}, err
//	}
//
//	// ClusterRoleBinding already exists - don't requeue
//	reqLogger.Info("Skip reconcile: Khulnasoft ClusterRoleBinding Exists", "ClusterRoleBinding.Namespace", found.Namespace, "ClusterRole.Name", found.Name)
//	return reconcile.Result{Requeue: true}, nil
//}
