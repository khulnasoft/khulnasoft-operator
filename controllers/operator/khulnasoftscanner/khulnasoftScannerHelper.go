package khulnasoftscanner

import (
	"fmt"
	"os"

	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ScannerParameters struct {
	Scanner *v1alpha1.KhulnasoftScanner
}

type KhulnasoftScannerHelper struct {
	Parameters ScannerParameters
}

func newKhulnasoftScannerHelper(cr *v1alpha1.KhulnasoftScanner) *KhulnasoftScannerHelper {
	params := ScannerParameters{
		Scanner: cr,
	}

	return &KhulnasoftScannerHelper{
		Parameters: params,
	}
}

func (as *KhulnasoftScannerHelper) CreateConfigMap(cr *v1alpha1.KhulnasoftScanner) *corev1.ConfigMap {

	labels := map[string]string{
		"app":                   "khulnasoft-scanner-config",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
	}

	annotations := map[string]string{
		"description": "Deploy Khulnasoft khulnasoft-csp-scanner ConfigMap",
	}

	data := map[string]string{
		"KHULNASOFT_SERVER": cr.Spec.Login.Host,
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.ScannerConfigMapName,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: data,
	}
	return configMap
}

func (as *KhulnasoftScannerHelper) CreateTokenSecret(cr *v1alpha1.KhulnasoftScanner) *corev1.Secret {
	labels := map[string]string{
		"app":                   cr.Name + "-requirments",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
	}
	annotations := map[string]string{
		"description": "Khulnasoft Scanner username and password",
	}
	token := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.ScannerSecretName,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"KHULNASOFT_SCANNER_USERNAME": []byte(cr.Spec.Login.Username),
			"KHULNASOFT_SCANNER_PASSWORD": []byte(cr.Spec.Login.Password),
		},
	}

	return token
}

func (as *KhulnasoftScannerHelper) newDeployment(cr *v1alpha1.KhulnasoftScanner) *appsv1.Deployment {
	pullPolicy, registry, repository, tag := extra.GetImageData("scanner", cr.Spec.Infrastructure.Version, cr.Spec.ScannerService.ImageData, cr.Spec.Common.AllowAnyVersion)

	image := os.Getenv("RELATED_IMAGE_SCANNER")
	if image == "" {
		image = fmt.Sprintf("%s/%s:%s", registry, repository, tag)
	}
	userarg := []string{
		"--user",
		"$(KHULNASOFT_SCANNER_USERNAME)",
		"--password",
		"$(KHULNASOFT_SCANNER_PASSWORD)",
	}
	tokenarg := []string{
		"--token",
		"$(KHULNASOFT_TOKEN)",
	}

	labels := map[string]string{
		"app":                   cr.Name + "-scanner",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":  "scanner",
	}

	annotations := map[string]string{
		"description":       "Deploy the khulnasoft scanner",
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
			Name:        fmt.Sprintf(consts.ScannerDeployName, cr.Name),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: extra.Int32Ptr(int32(cr.Spec.ScannerService.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                   cr.Name + "-scanner",
					"deployedby":            "khulnasoft-operator",
					"khulnasoftoperator_cr": cr.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   fmt.Sprintf(consts.ScannerDeployName, cr.Name),
					Annotations: map[string]string{
						"ConfigMapChecksum": cr.Spec.ConfigMapChecksum,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Spec.Infrastructure.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            "khulnasoft-scanner",
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Env: []corev1.EnvVar{
								{
									Name: "KHULNASOFT_SCANNER_LOGICAL_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: consts.ScannerSecretName,
										},
									},
								},
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: consts.ScannerConfigMapName,
										},
									},
								},
							},
							Args: []string{
								"daemon",
								"--host",
								"$(KHULNASOFT_SERVER)",
							},
							Ports: []corev1.ContainerPort{
								{
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	if cr.Spec.ScannerService.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *cr.Spec.ScannerService.Resources
	}

	if cr.Spec.ScannerService.LivenessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].LivenessProbe = cr.Spec.ScannerService.LivenessProbe
	}

	if cr.Spec.ScannerService.ReadinessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = cr.Spec.ScannerService.ReadinessProbe
	}

	if cr.Spec.ScannerService.NodeSelector != nil {
		if len(cr.Spec.ScannerService.NodeSelector) > 0 {
			deployment.Spec.Template.Spec.NodeSelector = cr.Spec.ScannerService.NodeSelector
		}
	}

	if cr.Spec.ScannerService.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity = cr.Spec.ScannerService.Affinity
	}

	if cr.Spec.ScannerService.Tolerations != nil {
		if len(cr.Spec.ScannerService.Tolerations) > 0 {
			deployment.Spec.Template.Spec.Tolerations = cr.Spec.ScannerService.Tolerations
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

	if cr.Spec.ScannerService.VolumeMounts != nil {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, cr.Spec.ScannerService.VolumeMounts...)
	}

	if cr.Spec.ScannerService.Volumes != nil {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, cr.Spec.ScannerService.Volumes...)
	}

	if len(cr.Spec.Login.Token) != 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "KHULNASOFT_TOKEN", Value: cr.Spec.Login.Token})
		deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args, tokenarg...)
	} else {
		deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args, userarg...)
	}

	if cr.Spec.Login.Insecure {
		deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args, "--no-verify")
	}

	return deployment
}
