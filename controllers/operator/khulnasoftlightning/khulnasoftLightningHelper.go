package khulnasoftlightning

import (
	"fmt"
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"strconv"
	"strings"
)

// EnforcerParameters
type LightningParameters struct {
	Lightning *v1alpha1.KhulnasoftLightning
}

// KhulnasoftEnforcerHelper
type KhulnasoftLightningHelper struct {
	Parameters LightningParameters
}

func newKhulnasoftLightningHelper(cr *v1alpha1.KhulnasoftLightning) *KhulnasoftLightningHelper {
	params := LightningParameters{
		Lightning: cr,
	}

	return &KhulnasoftLightningHelper{
		Parameters: params,
	}
}

func (lightning *KhulnasoftLightningHelper) newKhulnasoftKubeEnforcer(cr *v1alpha1.KhulnasoftLightning) *v1alpha1.KhulnasoftKubeEnforcer {
	// Step 1: Check if cr or cr.Spec is nil
	if cr == nil {
		return nil
	}

	// Step 2: Check if cr.Spec.KubeEnforcer is nil
	if cr.Spec.KubeEnforcer == nil {
		return nil
	}

	registry := consts.Registry
	if cr.Spec.KubeEnforcer.RegistryData != nil {
		if len(cr.Spec.KubeEnforcer.RegistryData.URL) > 0 {
			registry = cr.Spec.KubeEnforcer.RegistryData.URL
		}
	}
	tag := consts.LatestVersion
	if cr.Spec.KubeEnforcer.Infrastructure.Version != "" {
		tag = cr.Spec.KubeEnforcer.Infrastructure.Version
	}

	resources, err := yamlToResourceRequirements(consts.LightningKubeEnforcerResources)
	if err != nil {
		panic(err)
	}
	if cr.Spec.KubeEnforcer.KubeEnforcerService.Resources != nil {
		resources = cr.Spec.KubeEnforcer.KubeEnforcerService.Resources
	}

	sbResources, err := yamlToResourceRequirements(consts.LightningStarboardResources)
	if err != nil {
		panic(err)
	}
	if cr.Spec.KubeEnforcer.DeployStarboard.Resources != nil {
		sbResources = cr.Spec.KubeEnforcer.DeployStarboard.Resources
	}

	labels := map[string]string{
		"app":                   cr.Name + "-lightning",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":  "kubeenforcer",
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft KubeEnforcer",
	}

	KhulnasoftStarboardDetails := v1alpha1.KhulnasoftStarboardDetails{
		AllowAnyVersion: true,
		Infrastructure: &v1alpha1.KhulnasoftInfrastructure{
			Version:        consts.StarboardVersion,
			ServiceAccount: "starboard-operator",
		},
		Config: v1alpha1.KhulnasoftStarboardConfig{
			ImagePullSecret: "starboard-registry",
		},
		StarboardService: &v1alpha1.KhulnasoftService{
			Replicas: 1,
			ImageData: &v1alpha1.KhulnasoftImage{
				Registry:   "docker.io/khulnasoft",
				Repository: "starboard-operator",
				PullPolicy: "IfNotPresent",
			},
			Resources: sbResources,
		},
	}
	khulnasoftKubeEnf := &v1alpha1.KhulnasoftKubeEnforcer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.khulnasoft.com/v1alpha1",
			Kind:       "KhulnasoftKubeEnforcer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KhulnasoftKubeEnforcerSpec{
			Config: v1alpha1.KhulnasoftKubeEnforcerConfig{
				GatewayAddress:  cr.Spec.Global.GatewayAddress,
				ClusterName:     cr.Spec.Global.ClusterName,
				ImagePullSecret: cr.Spec.Common.ImagePullSecret,
			},
			Token:                  cr.Spec.KubeEnforcer.Token,
			EnforcerUpdateApproved: cr.Spec.KubeEnforcer.EnforcerUpdateApproved,
			AllowAnyVersion:        cr.Spec.KubeEnforcer.AllowAnyVersion,
			ImageData: &v1alpha1.KhulnasoftImage{
				Registry:   registry,
				Repository: "kube-enforcer",
				Tag:        tag,
				PullPolicy: "Always",
			},

			KubeEnforcerService: &v1alpha1.KhulnasoftService{
				Resources: resources,
			},

			DeployStarboard: &KhulnasoftStarboardDetails,
		},
	}

	return khulnasoftKubeEnf
}

func (lightning *KhulnasoftLightningHelper) newKhulnasoftEnforcer(cr *v1alpha1.KhulnasoftLightning) *v1alpha1.KhulnasoftEnforcer {
	if cr == nil || cr.Spec.Enforcer == nil || cr.Spec.Global == nil || cr.Spec.Global.GatewayAddress == "" {
		return nil
	}

	gwParts := strings.Split(cr.Spec.Global.GatewayAddress, ":")
	if len(gwParts) < 2 {
		return nil
	}

	gatewayHost := gwParts[0]
	gatewayPort, err := strconv.ParseInt(gwParts[1], 10, 64)
	if err != nil {
		panic(err)
	}

	registry := consts.Registry
	if cr.Spec.Enforcer.EnforcerService != nil && cr.Spec.Enforcer.EnforcerService.ImageData != nil {
		if len(cr.Spec.Enforcer.EnforcerService.ImageData.Registry) > 0 {
			registry = cr.Spec.Enforcer.EnforcerService.ImageData.Registry
		}
	}
	tag := consts.LatestVersion
	if cr.Spec.Enforcer.Infrastructure != nil && cr.Spec.Enforcer.Infrastructure.Version != "" {
		tag = cr.Spec.Enforcer.Infrastructure.Version
	}

	resources, err := yamlToResourceRequirements(consts.LightningEnforcerResources)
	if err != nil {
		panic(err)
	}
	if cr.Spec.Enforcer.EnforcerService != nil && cr.Spec.Enforcer.EnforcerService.Resources != nil {
		resources = cr.Spec.Enforcer.EnforcerService.Resources
	}

	labels := map[string]string{
		"app":                   cr.Name + "-enforcer",
		"deployedby":            "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":  "enforcer",
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Enforcer",
	}

	khulnasoftenf := &v1alpha1.KhulnasoftEnforcer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.khulnasoft.com/v1alpha1",
			Kind:       "KhulnasoftEnforcer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KhulnasoftEnforcerSpec{
			Infrastructure: cr.Spec.Enforcer.Infrastructure,
			Common:         cr.Spec.Common,
			Gateway: &v1alpha1.KhulnasoftGatewayInformation{
				Host: gatewayHost,
				Port: gatewayPort,
			},
			Token: cr.Spec.Enforcer.Token,
			Secret: &v1alpha1.KhulnasoftSecret{
				Name: cr.Spec.Enforcer.Secret.Name,
				Key:  cr.Spec.Enforcer.Secret.Key,
			},
			EnforcerService: &v1alpha1.KhulnasoftService{
				ImageData: &v1alpha1.KhulnasoftImage{
					Registry:   registry,
					Repository: "enforcer",
					Tag:        tag,
					PullPolicy: "Always",
				},
				Resources: resources,
			},
			RunAsNonRoot:           cr.Spec.Enforcer.RunAsNonRoot,
			EnforcerUpdateApproved: cr.Spec.Enforcer.EnforcerUpdateApproved,
		},
	}

	return khulnasoftenf
}

func yamlToResourceRequirements(yamlString string) (*v1.ResourceRequirements, error) {
	var yamlData map[string]map[string]map[string]string

	err := yaml.Unmarshal([]byte(yamlString), &yamlData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	limits := make(map[v1.ResourceName]resource.Quantity)
	requests := make(map[v1.ResourceName]resource.Quantity)

	for k, v := range yamlData["resources"]["limits"] {
		limits[v1.ResourceName(k)] = resource.MustParse(v)
	}
	for k, v := range yamlData["resources"]["requests"] {
		requests[v1.ResourceName(k)] = resource.MustParse(v)
	}

	resourceRequirements := &v1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}

	return resourceRequirements, nil
}
