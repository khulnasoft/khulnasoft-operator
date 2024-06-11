package khulnasoftenforcer

import (
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnforcerParameters :
type EnforcerParameters struct {
	Enforcer *v1alpha1.KhulnasoftEnforcer
}

// KhulnasoftEnforcerHelper :
type KhulnasoftEnforcerHelper struct {
	Parameters EnforcerParameters
}

func newKhulnasoftEnforcerHelper(cr *v1alpha1.KhulnasoftEnforcer) *KhulnasoftEnforcerHelper {
	params := EnforcerParameters{
		Enforcer: cr,
	}

	return &KhulnasoftEnforcerHelper{
		Parameters: params,
	}
}

// CreateTokenSecret : Create Enforcer Token Secret For The Enforcer connection to the khulnasoft csp environment
func (enf *KhulnasoftEnforcerHelper) CreateTokenSecret(cr *v1alpha1.KhulnasoftEnforcer) *corev1.Secret {
	labels := map[string]string{
		"app":                cr.Name + "-requirments",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftsecoperator_cr": cr.Name,
	}
	annotations := map[string]string{
		"description": "Secret for khulnasoft database password",
	}
	token := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf(consts.EnforcerTokenSecretName, cr.Name),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			consts.EnforcerTokenSecretKey: []byte(cr.Spec.Token),
		},
	}

	return token
}

func (enf *KhulnasoftEnforcerHelper) CreateConfigMap(cr *v1alpha1.KhulnasoftEnforcer) *corev1.ConfigMap {

	labels := map[string]string{
		"app":                "khulnasoft-csp-enforcer",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftsecoperator_cr": cr.Name,
	}

	annotations := map[string]string{
		"description": "Deploy Khulnasoft khulnasoft-csp-enforcer ConfigMap",
	}

	data := map[string]string{
		"KHULNASOFT_HEALTH_MONITOR_ENABLED": "true",
		"KHULNASOFT_INSTALL_PATH":           "/var/lib/khulnasoftsec",
		"KHULNASOFT_LOGICAL_NAME":           "",
		"KHULNASOFT_SERVER":                 fmt.Sprintf("%s:%d", cr.Spec.Gateway.Host, cr.Spec.Gateway.Port),
		"RESTART_CONTAINERS":          "no",
		"KHULNASOFT_EXPRESS_MODE":           "false",
	}

	if cr.Spec.KhulnasoftExpressMode {
		data["KHULNASOFT_EXPRESS_MODE"] = "true"
	}

	if len(cr.Spec.RhcosVersion) != 0 {
		data["KHULNASOFT_OCP_FULL_VERSION"] = cr.Spec.RhcosVersion
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.EnforcerConfigMapName,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: data,
	}
	return configMap
}

// CreateDaemonSet :
func (enf *KhulnasoftEnforcerHelper) CreateDaemonSet(cr *v1alpha1.KhulnasoftEnforcer) *appsv1.DaemonSet {
	pullPolicy, registry, repository, tag := extra.GetImageData("enforcer", cr.Spec.Infrastructure.Version, cr.Spec.EnforcerService.ImageData, cr.Spec.Common.AllowAnyVersion)

	image := os.Getenv("RELATED_IMAGE_ENFORCER")
	if image == "" {
		image = fmt.Sprintf("%s/%s:%s", registry, repository, tag)
	}

	labels := map[string]string{
		"app":                cr.Name + "-requirments",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftsecoperator_cr": cr.Name,
		"khulnasoft.component":     "enforcer",
	}
	annotations := map[string]string{
		"description":       "Deploy khulnasoft Enforcer",
		"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
	}

	privileged := true

	if cr.Spec.RunAsNonRoot {
		privileged = false
	}

	if !privileged {
		annotations["container.apparmor.security.beta.kubernetes.io/khulnasoft-agent"] = "unconfined"
	}

	envVars := enf.getEnvVars(cr)

	envFromSource := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: consts.EnforcerConfigMapName,
				},
			},
		},
	}

	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf(consts.EnforcerDeamonsetName, cr.Name),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                cr.Name + "-requirments",
					"deployedby":         "khulnasoft-operator",
					"khulnasoftsecoperator_cr": cr.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   fmt.Sprintf(consts.EnforcerDeamonsetName, cr.Name),
					Annotations: map[string]string{
						"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Spec.Infrastructure.ServiceAccount,
					HostPID:            true,
					RestartPolicy:      corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:            "khulnasoft-enforcer",
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "var-run",
									MountPath: "/var/run",
								},
								{
									Name:      "dev",
									MountPath: "/dev",
								},
								{
									Name:      "sys",
									MountPath: "/host/sys",
									ReadOnly:  true,
								},
								{
									Name:      "proc",
									MountPath: "/host/proc",
									ReadOnly:  true,
								},
								{
									Name:      "etc",
									MountPath: "/host/etc",
									ReadOnly:  true,
								},
								{
									Name:      "khulnasoftsec",
									MountPath: "/host/opt/khulnasoftsec",
									ReadOnly:  true,
								},
								{
									Name:      "khulnasoftsec-tmp",
									MountPath: "/opt/khulnasoftsec/tmp",
								},
								{
									Name:      "khulnasoftsec-audit",
									MountPath: "/opt/khulnasoftsec/audit",
								},
								{
									Name:      "khulnasoftsec-data",
									MountPath: "/data",
								},
							},
							Env:     envVars,
							EnvFrom: envFromSource,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8096),
										},
										Scheme: "HTTP",
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readinessz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8096),
										},
										Scheme: "HTTP",
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "var-run",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run",
								},
							},
						},
						{
							Name: "dev",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "proc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/proc",
								},
							},
						},
						{
							Name: "etc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc",
								},
							},
						},
						{
							Name: "khulnasoftsec",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/khulnasoftsec",
								},
							},
						},
						{
							Name: "khulnasoftsec-tmp",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/khulnasoftsec/tmp",
								},
							},
						},
						{
							Name: "khulnasoftsec-audit",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/khulnasoftsec/audit",
								},
							},
						},
						{
							Name: "khulnasoftsec-data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/khulnasoftsec/data",
								},
							},
						},
					},
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(1),
					},
				},
			},
		},
	}

	if cr.Spec.EnforcerService.Resources != nil {
		ds.Spec.Template.Spec.Containers[0].Resources = *cr.Spec.EnforcerService.Resources
	}

	if cr.Spec.EnforcerService.LivenessProbe != nil {
		ds.Spec.Template.Spec.Containers[0].LivenessProbe = cr.Spec.EnforcerService.LivenessProbe
	}

	if cr.Spec.EnforcerService.ReadinessProbe != nil {
		ds.Spec.Template.Spec.Containers[0].ReadinessProbe = cr.Spec.EnforcerService.ReadinessProbe
	}

	if cr.Spec.EnforcerService.NodeSelector != nil {
		if len(cr.Spec.EnforcerService.NodeSelector) > 0 {
			ds.Spec.Template.Spec.NodeSelector = cr.Spec.EnforcerService.NodeSelector
		}
	}

	if privileged {
		ds.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged: &privileged,
		}
	} else {
		ds.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged: &privileged,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"SYS_ADMIN",
					"NET_ADMIN",
					"NET_RAW",
					"SYS_PTRACE",
					"KILL",
					"MKNOD",
					"SETGID",
					"SETUID",
					"SYS_MODULE",
					"AUDIT_CONTROL",
					"SYSLOG",
					"SYS_CHROOT",
					"SYS_RESOURCE",
					"IPC_LOCK",
				},
			},
		}
	}

	if cr.Spec.Common != nil {
		if len(cr.Spec.Common.ImagePullSecret) != 0 {
			ds.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				corev1.LocalObjectReference{
					Name: cr.Spec.Common.ImagePullSecret,
				},
			}
		}
	}

	if cr.Spec.EnforcerService.VolumeMounts != nil {
		ds.Spec.Template.Spec.Containers[0].VolumeMounts = append(ds.Spec.Template.Spec.Containers[0].VolumeMounts, cr.Spec.EnforcerService.VolumeMounts...)
	}

	if cr.Spec.EnforcerService.Volumes != nil {
		ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, cr.Spec.EnforcerService.Volumes...)
	}

	if cr.Spec.EnforcerService.Tolerations != nil {
		ds.Spec.Template.Spec.Tolerations = cr.Spec.EnforcerService.Tolerations
	}

	if cr.Spec.EnforcerService.Affinity != nil {
		ds.Spec.Template.Spec.Affinity = cr.Spec.EnforcerService.Affinity
	}

	if cr.Spec.Mtls {
		mtlsKhulnasoftEnforcerVolumeMount := []corev1.VolumeMount{
			{
				Name:      "khulnasoft-grpc-enforcer",
				MountPath: "/opt/khulnasoftsec/ssl",
			},
		}

		secretVolumeSource := corev1.SecretVolumeSource{
			SecretName: "khulnasoft-grpc-enforcer",
		}

		mtlsKhulnasoftEnforcerVolume := []corev1.Volume{
			{
				Name: "khulnasoft-grpc-enforcer",
				VolumeSource: corev1.VolumeSource{
					Secret: &secretVolumeSource,
				},
			},
		}
		ds.Spec.Template.Spec.Containers[0].VolumeMounts = append(ds.Spec.Template.Spec.Containers[0].VolumeMounts, mtlsKhulnasoftEnforcerVolumeMount...)
		ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, mtlsKhulnasoftEnforcerVolume...)
	}

	return ds
}

func (ebf *KhulnasoftEnforcerHelper) getEnvVars(cr *v1alpha1.KhulnasoftEnforcer) []corev1.EnvVar {
	result := []corev1.EnvVar{
		{
			Name: "KHULNASOFT_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "spec.nodeName",
				},
			},
		},
		{
			Name: "KHULNASOFT_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cr.Spec.Secret.Name,
					},
					Key: cr.Spec.Secret.Key,
				},
			},
		},
	}

	if cr.Spec.Mtls {
		mtlsEnforcerEnv := []corev1.EnvVar{
			{
				Name:  "KHULNASOFT_PRIVATE_KEY",
				Value: "/opt/khulnasoftsec/ssl/khulnasoft_enforcer.key",
			},
			{
				Name:  "KHULNASOFT_PUBLIC_KEY",
				Value: "/opt/khulnasoftsec/ssl/khulnasoft_enforcer.crt",
			},
			{
				Name:  "KHULNASOFT_ROOT_CA",
				Value: "/opt/khulnasoftsec/ssl/rootCA.crt",
			},
		}
		result = append(result, mtlsEnforcerEnv...)
	}

	if cr.Spec.Envs != nil {
		for _, env := range cr.Spec.Envs {
			result = extra.AppendEnvVar(result, env)
		}
	}

	return result
}
