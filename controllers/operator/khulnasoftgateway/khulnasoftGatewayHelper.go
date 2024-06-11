package khulnasoftgateway

import (
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/controllers/common"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/k8s/services"
	"os"

	routev1 "github.com/openshift/api/route/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type GatewayParameters struct {
	Gateway *v1alpha1.KhulnasoftGateway
}

type KhulnasoftGatewayHelper struct {
	Parameters GatewayParameters
}

func newKhulnasoftGatewayHelper(cr *v1alpha1.KhulnasoftGateway) *KhulnasoftGatewayHelper {
	params := GatewayParameters{
		Gateway: cr,
	}

	return &KhulnasoftGatewayHelper{
		Parameters: params,
	}
}

func (gw *KhulnasoftGatewayHelper) newDeployment(cr *v1alpha1.KhulnasoftGateway) *appsv1.Deployment {
	pullPolicy, registry, repository, tag := extra.GetImageData("gateway", cr.Spec.Infrastructure.Version, cr.Spec.GatewayService.ImageData, cr.Spec.Common.AllowAnyVersion)

	image := os.Getenv("RELATED_IMAGE_GATEWAY")
	if image == "" {
		image = fmt.Sprintf("%s/%s:%s", registry, repository, tag)
	}

	labels := map[string]string{
		"app":                      cr.Name + "-gateway",
		"deployedby":               "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"type":                     "khulnasoft-gateway",
		"khulnasoft.component":     "gateway",
	}
	annotations := map[string]string{
		"description": "Deploy the khulnasoft gateway server",
	}

	envVars := gw.getEnvVars(cr)

	privileged := true

	if cr.Spec.RunAsNonRoot {
		privileged = false
	}

	envFromSource := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: consts.ServerConfigMapName,
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf(consts.GatewayDeployName, cr.Name),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: extra.Int32Ptr(int32(cr.Spec.GatewayService.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                      cr.Name + "-gateway",
					"deployedby":               "khulnasoft-operator",
					"khulnasoftoperator_cr": cr.Name,
					"type":                     "khulnasoft-gateway",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Spec.Infrastructure.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            "khulnasoft-gateway",
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Ports: []corev1.ContainerPort{
								{
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 3622,
								},
								{
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8443,
								},
							},
							Env:     envVars,
							EnvFrom: envFromSource,
							LivenessProbe: &corev1.Probe{
								FailureThreshold: 3,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8082),
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
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8443),
										},
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
							},
						},
					},
				},
			},
		},
	}

	if cr.Spec.GatewayService.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *cr.Spec.GatewayService.Resources
	}

	if cr.Spec.GatewayService.LivenessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].LivenessProbe = cr.Spec.GatewayService.LivenessProbe
	}

	if cr.Spec.GatewayService.ReadinessProbe != nil {
		deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = cr.Spec.GatewayService.ReadinessProbe
	}

	if cr.Spec.GatewayService.NodeSelector != nil {
		if len(cr.Spec.GatewayService.NodeSelector) > 0 {
			deployment.Spec.Template.Spec.NodeSelector = cr.Spec.GatewayService.NodeSelector
		}
	}

	if cr.Spec.GatewayService.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity = cr.Spec.GatewayService.Affinity
	}

	if cr.Spec.GatewayService.Tolerations != nil {
		if len(cr.Spec.GatewayService.Tolerations) > 0 {
			deployment.Spec.Template.Spec.Tolerations = cr.Spec.GatewayService.Tolerations
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

	if cr.Spec.GatewayService.VolumeMounts != nil {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, cr.Spec.GatewayService.VolumeMounts...)
	}

	if cr.Spec.GatewayService.Volumes != nil {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, cr.Spec.GatewayService.Volumes...)
	}

	if cr.Spec.Mtls {
		mtlsKhulnasoftGatewayVolumeMount := []corev1.VolumeMount{
			{
				Name:      "khulnasoft-grpc-gateway",
				MountPath: "/opt/khulnasoft/ssl",
				ReadOnly:  true,
			},
		}

		secretVolumeSource := corev1.SecretVolumeSource{
			SecretName: "khulnasoft-grpc-gateway",
			Items: []corev1.KeyToPath{
				{
					Key:  "khulnasoft_gateway.crt",
					Path: "cert.pem",
				},
				{
					Key:  "khulnasoft_gateway.key",
					Path: "key.pem",
				},
				{
					Key:  "rootCA.crt",
					Path: "ca.pem",
				},
			},
		}

		mtlsKhulnasoftGatewayVolume := []corev1.Volume{
			{
				Name: "khulnasoft-grpc-gateway",
				VolumeSource: corev1.VolumeSource{
					Secret: &secretVolumeSource,
				},
			},
		}
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, mtlsKhulnasoftGatewayVolumeMount...)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, mtlsKhulnasoftGatewayVolume...)
	}

	return deployment
}

func (gw *KhulnasoftGatewayHelper) getEnvVars(cr *v1alpha1.KhulnasoftGateway) []corev1.EnvVar {
	envsHelper := common.NewKhulnasoftEnvsHelper(cr.Spec.Infrastructure, cr.Spec.Common, cr.Spec.ExternalDb, cr.Name, cr.Spec.AuditDB)
	result, _ := envsHelper.GetDbEnvVars()

	if cr.Spec.Envs != nil {
		for _, env := range cr.Spec.Envs {
			result = extra.AppendEnvVar(result, env)
		}
	}

	return result
}

func (gw *KhulnasoftGatewayHelper) newService(cr *v1alpha1.KhulnasoftGateway) *corev1.Service {
	selectors := map[string]string{
		"app": fmt.Sprintf("%s-gateway", cr.Name),
	}

	ports := []corev1.ServicePort{
		{
			Port:       3622,
			TargetPort: intstr.FromInt(3622),
			Name:       "khulnasoft-gate",
		},
		{
			Port:       8443,
			TargetPort: intstr.FromInt(8443),
			Name:       "khulnasoft-gate-ssl",
		},
	}

	service := services.CreateService(cr.Name,
		cr.Namespace,
		fmt.Sprintf(consts.GatewayServiceName, cr.Name),
		fmt.Sprintf("%s-gateway", cr.Name),
		"Service for khulnasoft gateway components",
		cr.Spec.GatewayService.ServiceType,
		selectors,
		ports)

	return service
}

func (gw *KhulnasoftGatewayHelper) newRoute(cr *v1alpha1.KhulnasoftGateway) *routev1.Route {

	gwServiceName := fmt.Sprintf(consts.GatewayServiceName, cr.Name)

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gwServiceName,
			Namespace: cr.Namespace,
		},
		Spec: routev1.RouteSpec{
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
				Termination:                   routev1.TLSTerminationPassthrough,
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: gwServiceName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(8443),
			},
		},
	}
}
