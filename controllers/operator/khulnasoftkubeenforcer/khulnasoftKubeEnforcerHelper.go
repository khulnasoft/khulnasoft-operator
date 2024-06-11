package khulnasoftkubeenforcer

import (
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/khulnasoft/v1alpha1"
	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	rbac2 "github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/rbac"
	"os"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	WebhookTimeout int32 = 5
)

// EnforcerParameters :
type KubeEnforcerParameters struct {
	KubeEnforcer *operatorv1alpha1.KhulnasoftKubeEnforcer
}

// KhulnasoftEnforcerHelper :
type KhulnasoftKubeEnforcerHelper struct {
	Parameters KubeEnforcerParameters
}

func newKhulnasoftKubeEnforcerHelper(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) *KhulnasoftKubeEnforcerHelper {
	params := KubeEnforcerParameters{
		KubeEnforcer: cr,
	}

	return &KhulnasoftKubeEnforcerHelper{
		Parameters: params,
	}
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKubeEnforcerClusterRole(name string, namespace string) *rbacv1.ClusterRole {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"pods",
				"nodes",
				"namespaces",
				"deployments",
				"statefulsets",
				"jobs",
				"cronjobs",
				"daemonsets",
				"replicasets",
				"replicationcontrollers",
				"clusterroles",
				"clusterrolebindings",
				"componentstatuses",
				"services",
			},
			Verbs: []string{
				"get", "list", "watch",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"secrets",
			},
			Verbs: []string{
				"get", "list", "watch", "update", "create", "delete",
			},
		},
		{
			APIGroups: []string{
				"khulnasoft.github.io",
			},
			Resources: []string{
				"configauditreports",
				"clusterconfigauditreports",
			},
			Verbs: []string{
				"get", "list", "watch",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"configmaps",
			},
			Verbs: []string{
				"get", "list", "watch", "update", "create",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"roles",
				"rolebindings",
				"clusterroles",
				"clusterrolebindings",
			},
			Verbs: []string{
				"get", "list", "watch",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"customresourcedefinitions",
			},
			Verbs: []string{
				"get", "list", "watch",
			},
		},
	}

	crole := rbac2.CreateClusterRole(name, namespace, "khulnasoft-kube-enforcer", fmt.Sprintf("%s-rbac", "khulnasoft-ke"), "Deploy Khulnasoft Discovery Cluster Role", rules)

	return crole
}

// CreateServiceAccount Create new service account
func (enf *KhulnasoftKubeEnforcerHelper) CreateKEServiceAccount(cr, namespace, app, name string) *corev1.ServiceAccount {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Service account for khulnasoft kube-enforcer",
	}
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	return sa
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateClusterRoleBinding(cr, namespace, name, app, sa, clusterrole string) *rbacv1.ClusterRoleBinding {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Cluster Role Binding",
	}
	crb := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterrole,
		},
	}

	return crb
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKubeEnforcerRole(cr, namespace, name, app string) *rbacv1.Role {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"pods/log",
			},
			Verbs: []string{
				"get", "list", "watch",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"jobs",
			},
			Verbs: []string{
				"create", "delete",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"leases",
			},
			Verbs: []string{
				"get", "list", "create", "update",
			},
		},
		{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"pods",
			},
			Verbs: []string{
				"create", "delete",
			},
		},
	}
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description":              "KubeEnforcer Role",
		"openshift.io/description": "A user who can search and scan images from an OpenShift integrated registry.",
	}
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Rules: rules,
	}

	return role
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateRoleBinding(cr, namespace, name, app, sa, role string) *rbacv1.RoleBinding {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Cluster Role Binding",
	}
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role,
		},
	}

	return rb
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateValidatingWebhook(cr, namespace, name, app, keService string, caBundle []byte) *admissionv1.ValidatingWebhookConfiguration {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft ValidatingWebhookConfiguration",
	}
	rules := []admissionv1.RuleWithOperations{
		{
			Operations: []admissionv1.OperationType{
				admissionv1.Create,
				admissionv1.Update,
			},
			Rule: admissionv1.Rule{
				APIGroups: []string{
					"*",
				},
				APIVersions: []string{
					"*",
				},
				Resources: []string{
					"pods",
					"deployments",
					"replicasets",
					"replicationcontrollers",
					"statefulsets",
					"daemonsets",
					"jobs",
					"cronjobs",
					"configmaps",
					"services",
					"roles",
					"rolebindings",
					"clusterroles",
					"clusterrolebindings",
					"customresourcedefinitions",
				},
			},
		},
	}
	servicePort := int32(443)
	sideEffect := admissionv1.SideEffectClassNone
	failurePolicy := admissionv1.Ignore
	validWebhook := &admissionv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name:  "imageassurance.khulnasoft.com",
				Rules: rules,
				ClientConfig: admissionv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &admissionv1.ServiceReference{
						Namespace: namespace,
						Name:      keService,
						Port:      &servicePort,
					},
				},
				TimeoutSeconds:          extra.Int32Ptr(WebhookTimeout),
				SideEffects:             &sideEffect,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &failurePolicy,
			},
		},
	}

	return validWebhook
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateMutatingWebhook(cr, namespace, name, app, keService string, caBundle []byte) *admissionv1.MutatingWebhookConfiguration {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft MutatingWebhookConfiguration",
	}
	rules := []admissionv1.RuleWithOperations{
		{
			Operations: []admissionv1.OperationType{
				admissionv1.Create,
				admissionv1.Update,
			},
			Rule: admissionv1.Rule{
				APIGroups: []string{
					"*",
				},
				APIVersions: []string{
					"v1",
				},
				Resources: []string{
					"pods",
				},
			},
		},
	}
	mutatePath := "/mutate"
	servicePort := int32(443)
	sideEffect := admissionv1.SideEffectClassNone
	failurePolicy := admissionv1.Ignore
	mutateWebhook := &admissionv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				Name:  "microenforcer.khulnasoft.com",
				Rules: rules,
				ClientConfig: admissionv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &admissionv1.ServiceReference{
						Namespace: namespace,
						Name:      keService,
						Path:      &mutatePath,
						Port:      &servicePort,
					},
				},
				TimeoutSeconds:          extra.Int32Ptr(WebhookTimeout),
				SideEffects:             &sideEffect,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &failurePolicy,
			},
		},
	}

	return mutateWebhook
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKEConfigMap(cr, namespace, name, app, gwAddress, clusterName string, starboard bool) *corev1.ConfigMap {
	configMapData := map[string]string{
		"KHULNASOFT_ENABLE_CACHE":            "yes",
		"KHULNASOFT_CACHE_EXPIRATION_PERIOD": "60",
		"TLS_SERVER_CERT_FILEPATH":           "/certs/khulnasoft_ke.crt",
		"TLS_SERVER_KEY_FILEPATH":            "/certs/khulnasoft_ke.key",
		"KHULNASOFT_GATEWAY_SECURE_ADDRESS":  gwAddress,
		"KHULNASOFT_TLS_PORT":                "8443",
		"CLUSTER_NAME":                       clusterName,
	}
	if starboard {
		configMapData["KHULNASOFT_KAP_ADD_ALL_CONTROL"] = "true"
		configMapData["KHULNASOFT_WATCH_CONFIG_AUDIT_REPORT"] = "true"
	}

	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft KubeEnfocer ConfigMap",
	}
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: configMapData,
	}

	return configMap
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKETokenSecret(cr, namespace, name, app, token string) *corev1.Secret {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft KubeEnfocer token secret",
	}
	tokenSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string][]byte{
			"token": []byte(token),
		},
	}

	return tokenSecret
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKESSLSecret(cr, namespace, name, app string, secretKey, secretCert []byte) *corev1.Secret {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Kube Enforcer SSL certificates to communicate with Kube API server",
	}
	sslSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string][]byte{
			"khulnasoft_ke.key": secretKey,
			"khulnasoft_ke.crt": secretCert,
		},
	}

	return sslSecret
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKEService(cr, namespace, name, app string) *corev1.Service {
	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr,
	}
	annotations := map[string]string{
		"description": "Deploy Kube Enforcer Service",
	}
	selectors := map[string]string{
		"app": "khulnasoft-kube-enforcer",
	}

	ports := []corev1.ServicePort{
		{
			Port:       443,
			TargetPort: intstr.FromInt(8443),
		},
	}
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(enf.Parameters.KubeEnforcer.Spec.KubeEnforcerService.ServiceType),
			Selector: selectors,
			Ports:    ports,
		},
	}

	return service
}

func (enf *KhulnasoftKubeEnforcerHelper) CreateKEDeployment(cr *operatorv1alpha1.KhulnasoftKubeEnforcer, name, app, registry, tag, pullPolicy, repository string) *appsv1.Deployment {

	image := os.Getenv("RELATED_IMAGE_KUBE_ENFORCER")
	if image == "" {
		image = fmt.Sprintf("%s/%s:%s", registry, repository, tag)
	}

	labels := map[string]string{
		"app":                   app,
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":  "kubeenforcer",
	}
	annotations := map[string]string{
		"description":       "Deploy Kube Enforcer Deployment",
		"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
	}

	envVars := enf.getEnvVars(cr)
	selectors := map[string]string{
		"app": "khulnasoft-kube-enforcer",
	}

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 8443,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		},
	}
	runAsUser := int64(11431)
	runAsGroup := int64(11433)
	fsGroup := int64(11433)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			//Replicas: extra.Int32Ptr(int32(2)),
			Selector: &metav1.LabelSelector{
				MatchLabels: selectors,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectors,
					Annotations: map[string]string{
						"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &runAsUser,
						RunAsGroup: &runAsGroup,
						FSGroup:    &fsGroup,
					},
					ServiceAccountName: cr.Spec.Infrastructure.ServiceAccount,
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: cr.Spec.Config.ImagePullSecret,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kube-enforcer-ssl",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "kube-enforcer-ssl",
									Items: []corev1.KeyToPath{
										{
											Key:  "khulnasoft_ke.crt",
											Path: "khulnasoft_ke.crt",
										},
										{
											Key:  "khulnasoft_ke.key",
											Path: "khulnasoft_ke.key",
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "kube-enforcer",
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							LivenessProbe: &corev1.Probe{
								FailureThreshold: 3,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8080),
										},
										Scheme: "HTTP",
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold: 3,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8080),
										},
										Scheme: "HTTP",
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
							},
							Ports: ports,
							Env:   envVars,
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "khulnasoft-csp-kube-enforcer",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kube-enforcer-ssl",
									MountPath: "/certs",
								},
							},
						},
					},
				},
			},
		},
	}

	kubeEnforcerExtraData := enf.Parameters.KubeEnforcer.Spec.KubeEnforcerService

	if kubeEnforcerExtraData.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *kubeEnforcerExtraData.Resources
	}

	if kubeEnforcerExtraData.LivenessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].LivenessProbe = kubeEnforcerExtraData.LivenessProbe
	}

	if kubeEnforcerExtraData.ReadinessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = kubeEnforcerExtraData.ReadinessProbe
	}

	if kubeEnforcerExtraData.VolumeMounts != nil {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, kubeEnforcerExtraData.VolumeMounts...)
	}

	if kubeEnforcerExtraData.Volumes != nil {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, kubeEnforcerExtraData.Volumes...)
	}

	if enf.Parameters.KubeEnforcer.Spec.Envs != nil {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, enf.Parameters.KubeEnforcer.Spec.Envs...)
	}

	if cr.Spec.Mtls {
		mtlsKhulnasoftKubeEnforcerVolumeMount := []corev1.VolumeMount{
			{
				Name:      "khulnasoft-grpc-kube-enforcer",
				MountPath: "/opt/khulnasoft/ssl",
			},
		}

		secretVolumeSource := corev1.SecretVolumeSource{
			SecretName: "khulnasoft-grpc-kube-enforcer",
		}

		mtlsKhulnasoftKubeEnforcerVolume := []corev1.Volume{
			{
				Name: "khulnasoft-grpc-kube-enforcer",
				VolumeSource: corev1.VolumeSource{
					Secret: &secretVolumeSource,
				},
			},
		}
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, mtlsKhulnasoftKubeEnforcerVolumeMount...)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, mtlsKhulnasoftKubeEnforcerVolume...)
	}

	return deployment
}

func (ebf *KhulnasoftKubeEnforcerHelper) getEnvVars(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) []corev1.EnvVar {
	result := []corev1.EnvVar{
		{
			Name: "KHULNASOFT_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "khulnasoft-kube-enforcer-token",
					},
					Key: "token",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	if cr.Spec.Mtls {
		mtlsKubeEnforcerEnv := []corev1.EnvVar{
			{
				Name:  "KHULNASOFT_PRIVATE_KEY",
				Value: "/opt/khulnasoft/ssl/khulnasoft_kube-enforcer.key",
			},
			{
				Name:  "KHULNASOFT_PUBLIC_KEY",
				Value: "/opt/khulnasoft/ssl/khulnasoft_kube-enforcer.crt",
			},
			{
				Name:  "KHULNASOFT_ROOT_CA",
				Value: "/opt/khulnasoft/ssl/rootCA.crt",
			},
			{
				Name:  "KHULNASOFT_TLS_VERIFY",
				Value: "true",
			},
		}
		result = append(result, mtlsKubeEnforcerEnv...)
	}

	return result
}

// Starboard functions

func (ebf *KhulnasoftKubeEnforcerHelper) newStarboard(cr *operatorv1alpha1.KhulnasoftKubeEnforcer) *v1alpha1.KhulnasoftStarboard {

	_, registry, repository, tag := extra.GetImageData("kube-enforcer", cr.Spec.Infrastructure.Version, cr.Spec.KubeEnforcerService.ImageData, cr.Spec.AllowAnyVersion)

	labels := map[string]string{
		"app":                   cr.Name + "-kube-enforcer",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Starboard",
	}

	khulnasoftsb := &v1alpha1.KhulnasoftStarboard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "khulnasoft.github.io/v1alpha1",
			Kind:       "KhulnasoftStarboard",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KhulnasoftStarboardSpec{
			Infrastructure:                cr.Spec.DeployStarboard.Infrastructure,
			AllowAnyVersion:               cr.Spec.DeployStarboard.AllowAnyVersion,
			StarboardService:              cr.Spec.DeployStarboard.StarboardService,
			Config:                        cr.Spec.DeployStarboard.Config,
			RegistryData:                  cr.Spec.DeployStarboard.RegistryData,
			ImageData:                     cr.Spec.DeployStarboard.ImageData,
			Envs:                          cr.Spec.DeployStarboard.Envs,
			KubeEnforcerVersion:           fmt.Sprintf("%s/%s:%s", registry, repository, tag),
			LogDevMode:                    cr.Spec.DeployStarboard.LogDevMode,
			ConcurrentScanJobsLimit:       cr.Spec.DeployStarboard.ConcurrentScanJobsLimit,
			ScanJobRetryAfter:             cr.Spec.DeployStarboard.ScanJobRetryAfter,
			MetricsBindAddress:            cr.Spec.DeployStarboard.MetricsBindAddress,
			HealthProbeBindAddress:        cr.Spec.DeployStarboard.HealthProbeBindAddress,
			CisKubernetesBenchmarkEnabled: cr.Spec.DeployStarboard.CisKubernetesBenchmarkEnabled,
			VulnerabilityScannerEnabled:   cr.Spec.DeployStarboard.VulnerabilityScannerEnabled,
			BatchDeleteLimit:              cr.Spec.DeployStarboard.BatchDeleteLimit,
			BatchDeleteDelay:              cr.Spec.DeployStarboard.BatchDeleteLimit,
		},
	}
	return khulnasoftsb
}
