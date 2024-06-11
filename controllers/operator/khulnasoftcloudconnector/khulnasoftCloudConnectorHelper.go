package khulnasoftcloudconnector

import (
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
)

type CloudConnectorParameters struct {
	CloudConnector *v1alpha1.KhulnasoftCloudConnector
}

type KhulnasoftCloudConnectorHelper struct {
	Parameters CloudConnectorParameters
}

type test struct {
	Host string `json:"host,omitempty"`
}

type test1 struct {
	Tunnels []test `yaml:"tunnels"`
}

func newKhulnasoftCloudConnectorHelper(cr *v1alpha1.KhulnasoftCloudConnector) *KhulnasoftCloudConnectorHelper {
	params := CloudConnectorParameters{
		CloudConnector: cr,
	}

	return &KhulnasoftCloudConnectorHelper{
		Parameters: params,
	}
}

func (as *KhulnasoftCloudConnectorHelper) CreateConfigMap(cr *v1alpha1.KhulnasoftCloudConnector) *corev1.ConfigMap {

	labels := map[string]string{
		"app":                   "khulnasoft-cloud-connector-conf",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
	}

	annotations := map[string]string{
		"description": "Deploy Khulnasoft khulnasoft-csp-cloud-connector ConfigMap",
	}
	tunnels := "tunnels:\n"

	for _, tunnel := range cr.Spec.Tunnels {
		if tunnel.Region != "" {
			tunnels = tunnels + fmt.Sprintf("  - service:\n      type: %s\n      region: %s\n", tunnel.Type, tunnel.Region)
		} else {
			tunnels = tunnels + fmt.Sprintf("  - host: %s\n  - port: %s\n", tunnel.Host, tunnel.Port)
		}
	}

	data := map[string]string{
		"khulnasoft-tunnels-cloud-connector-config": tunnels,
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.CloudConnectorConfigMapName,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: data,
	}
	return configMap
}

func (as *KhulnasoftCloudConnectorHelper) CreateTokenSecret(cr *v1alpha1.KhulnasoftCloudConnector) *corev1.Secret {

	labels := map[string]string{
		"app":                   cr.Name + "-requirments",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
	}
	annotations := map[string]string{
		"description": "Khulnasoft CloudConnector username and password",
	}
	token := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.CloudConnectorSecretName,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte(cr.Spec.Login.Username),
			"password": []byte(cr.Spec.Login.Password),
		},
	}
	if len(cr.Spec.Login.Token) != 0 {
		token.Data["token"] = []byte(cr.Spec.Login.Token)
	}

	return token
}

func (as *KhulnasoftCloudConnectorHelper) newDeployment(cr *v1alpha1.KhulnasoftCloudConnector) *appsv1.Deployment {
	pullPolicy, registry, repository, tag := extra.GetImageData("khulnasoft-cloud-connector", cr.Spec.Infrastructure.Version, cr.Spec.CloudConnectorService.ImageData, cr.Spec.Common.AllowAnyVersion)
	cloudConnectorTerminationGracePeriodSeconds := int64(30)
	image := os.Getenv("RELATED_IMAGE_CLOUD_CONNECTOR")
	if image == "" {
		image = fmt.Sprintf("%s/%s:%s", registry, repository, tag)
	}

	labels := map[string]string{
		"app":                   cr.Name + "-cloud-connector",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":  "cloud-connector",
	}

	annotations := map[string]string{
		"description":       "Deploy the khulnasoft cloud-connector",
		"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
	}

	privileged := true

	if cr.Spec.RunAsNonRoot {
		privileged = false
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf(consts.CloudConnectorDeployName, cr.Name),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: extra.Int32Ptr(int32(cr.Spec.CloudConnectorService.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                   cr.Name + "-cloud-connector",
					"deployedby":            "khulnasoft-operator",
					"khulnasoftoperator_cr": cr.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   fmt.Sprintf(consts.CloudConnectorDeployName, cr.Name),
					Annotations: map[string]string{
						"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Spec.Infrastructure.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            "khulnasoft-cloud-connector",
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
							Env: []corev1.EnvVar{
								{
									Name:  "KHULNASOFT_SERVER",
									Value: cr.Spec.Login.Host,
								},
								{
									Name:  "KHULNASOFT_CLOUD_CONNECTOR_CONFIG_FILE_PATH",
									Value: "/etc/config/connector.yaml",
								},
								{
									Name: "KHULNASOFT_CLOUD_CONNECTOR_USERNAME",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "username",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: consts.CloudConnectorSecretName,
											},
										},
									},
								},
								{
									Name: "KHULNASOFT_CLOUD_CONNECTOR_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "password",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: consts.CloudConnectorSecretName,
											},
										},
									},
								},
								{
									Name:  "KHULNASOFT_CLOUD_CONNECTOR_HEALTH_PORT",
									Value: "8080",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8080),
										},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    10,
							},
							Ports: []corev1.ContainerPort{
								{
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/config",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: consts.CloudConnectorConfigMapName,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "khulnasoft-tunnels-cloud-connector-config",
											Path: "connector.yaml",
										},
									},
								},
							},
						},
					},
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 corev1.DefaultSchedulerName,
					TerminationGracePeriodSeconds: &cloudConnectorTerminationGracePeriodSeconds,
				},
			},
		},
	}

	if cr.Spec.CloudConnectorService.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *cr.Spec.CloudConnectorService.Resources
	}

	if cr.Spec.CloudConnectorService.LivenessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].LivenessProbe = cr.Spec.CloudConnectorService.LivenessProbe
	}

	if cr.Spec.CloudConnectorService.ReadinessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = cr.Spec.CloudConnectorService.ReadinessProbe
	}

	if cr.Spec.CloudConnectorService.NodeSelector != nil {
		if len(cr.Spec.CloudConnectorService.NodeSelector) > 0 {
			deployment.Spec.Template.Spec.NodeSelector = cr.Spec.CloudConnectorService.NodeSelector
		}
	}

	if cr.Spec.CloudConnectorService.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity = cr.Spec.CloudConnectorService.Affinity
	}

	if cr.Spec.CloudConnectorService.Tolerations != nil {
		if len(cr.Spec.CloudConnectorService.Tolerations) > 0 {
			deployment.Spec.Template.Spec.Tolerations = cr.Spec.CloudConnectorService.Tolerations
		}
	}

	if cr.Spec.Common != nil {
		if len(cr.Spec.Common.ImagePullSecret) != 0 {
			deployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				corev1.LocalObjectReference{
					Name: cr.Spec.Common.ImagePullSecret,
				},
			}
		}
	}

	if cr.Spec.RunAsNonRoot {
		runAsUser := int64(11431)
		runAsGroup := int64(11433)
		fsGroup := int64(11433)
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:    &runAsUser,
			RunAsGroup:   &runAsGroup,
			RunAsNonRoot: &cr.Spec.RunAsNonRoot,
			FSGroup:      &fsGroup,
		}
	}

	if cr.Spec.CloudConnectorService.VolumeMounts != nil {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, cr.Spec.CloudConnectorService.VolumeMounts...)
	}

	if cr.Spec.CloudConnectorService.Volumes != nil {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, cr.Spec.CloudConnectorService.Volumes...)
	}

	if len(cr.Spec.Login.Token) != 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "KHULNASOFT_CLOUD_CONNECTOR_TOKEN", ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				Key: "token",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: consts.CloudConnectorSecretName,
				},
			},
		}})
	}

	if cr.Spec.Login.Insecure {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "KHULNASOFT_TLS_VERIFY", Value: "0"})
	} else {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "KHULNASOFT_TLS_VERIFY", Value: "1"})
	}

	return deployment
}
