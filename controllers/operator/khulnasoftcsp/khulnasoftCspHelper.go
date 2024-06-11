package khulnasoftcsp

import (
	"fmt"

	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CspParameters struct {
	KhulnasoftCsp *v1alpha1.KhulnasoftCsp
}

type KhulnasoftCspHelper struct {
	Parameters CspParameters
}

func newKhulnasoftCspHelper(cr *v1alpha1.KhulnasoftCsp) *KhulnasoftCspHelper {
	params := CspParameters{
		KhulnasoftCsp: cr,
	}

	return &KhulnasoftCspHelper{
		Parameters: params,
	}
}

func (csp *KhulnasoftCspHelper) newKhulnasoftDatabase(cr *v1alpha1.KhulnasoftCsp) *v1alpha1.KhulnasoftDatabase {
	labels := map[string]string{
		"app":                cr.Name + "-csp",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":     "database",
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Database (not for production environments)",
	}
	khulnasoftdb := &v1alpha1.KhulnasoftDatabase{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.khulnasoft.com/v1alpha1",
			Kind:       "KhulnasoftDatabase",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KhulnasoftDatabaseSpec{
			Infrastructure: csp.Parameters.KhulnasoftCsp.Spec.Infrastructure,
			Common:         csp.Parameters.KhulnasoftCsp.Spec.Common,
			DbService:      csp.Parameters.KhulnasoftCsp.Spec.DbService,
			DiskSize:       csp.Parameters.KhulnasoftCsp.Spec.Common.DbDiskSize,
			RunAsNonRoot:   csp.Parameters.KhulnasoftCsp.Spec.RunAsNonRoot,
			AuditDB:        csp.Parameters.KhulnasoftCsp.Spec.AuditDB,
		},
	}

	return khulnasoftdb
}

func (csp *KhulnasoftCspHelper) newKhulnasoftGateway(cr *v1alpha1.KhulnasoftCsp) *v1alpha1.KhulnasoftGateway {
	labels := map[string]string{
		"app":                cr.Name + "-csp",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":     "gateway",
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Gateway",
	}
	khulnasoftgateway := &v1alpha1.KhulnasoftGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.khulnasoft.com/v1alpha1",
			Kind:       "KhulnasoftGateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KhulnasoftGatewaySpec{
			Infrastructure: csp.Parameters.KhulnasoftCsp.Spec.Infrastructure,
			Common:         csp.Parameters.KhulnasoftCsp.Spec.Common,
			GatewayService: csp.Parameters.KhulnasoftCsp.Spec.GatewayService,
			ExternalDb:     csp.Parameters.KhulnasoftCsp.Spec.ExternalDb,
			RunAsNonRoot:   csp.Parameters.KhulnasoftCsp.Spec.RunAsNonRoot,
			Envs:           csp.Parameters.KhulnasoftCsp.Spec.GatewayEnvs,
			AuditDB:        csp.Parameters.KhulnasoftCsp.Spec.AuditDB,
			Route:          csp.Parameters.KhulnasoftCsp.Spec.Route,
		},
	}

	return khulnasoftgateway
}

func (csp *KhulnasoftCspHelper) newKhulnasoftServer(cr *v1alpha1.KhulnasoftCsp) *v1alpha1.KhulnasoftServer {
	labels := map[string]string{
		"app":                cr.Name + "-csp",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":     "server",
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Server",
	}
	khulnasoftServer := &v1alpha1.KhulnasoftServer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.khulnasoft.com/v1alpha1",
			Kind:       "KhulnasoftServer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KhulnasoftServerSpec{
			Infrastructure: csp.Parameters.KhulnasoftCsp.Spec.Infrastructure,
			Common:         csp.Parameters.KhulnasoftCsp.Spec.Common,
			ServerService:  csp.Parameters.KhulnasoftCsp.Spec.ServerService,
			ExternalDb:     csp.Parameters.KhulnasoftCsp.Spec.ExternalDb,
			LicenseToken:   csp.Parameters.KhulnasoftCsp.Spec.LicenseToken,
			AdminPassword:  csp.Parameters.KhulnasoftCsp.Spec.AdminPassword,
			Enforcer:       csp.Parameters.KhulnasoftCsp.Spec.Enforcer,
			RunAsNonRoot:   csp.Parameters.KhulnasoftCsp.Spec.RunAsNonRoot,
			Envs:           csp.Parameters.KhulnasoftCsp.Spec.ServerEnvs,
			ConfigMapData:  csp.Parameters.KhulnasoftCsp.Spec.ServerConfigMapData,
			AuditDB:        csp.Parameters.KhulnasoftCsp.Spec.AuditDB,
			Route:          csp.Parameters.KhulnasoftCsp.Spec.Route,
		},
	}

	return khulnasoftServer
}

func (csp *KhulnasoftCspHelper) newKhulnasoftEnforcer(cr *v1alpha1.KhulnasoftCsp) *v1alpha1.KhulnasoftEnforcer {
	registry := consts.Registry
	if cr.Spec.RegistryData != nil {
		if len(cr.Spec.RegistryData.URL) > 0 {
			registry = cr.Spec.RegistryData.URL
		}
	}

	labels := map[string]string{
		"app":                cr.Name + "-csp",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":     "enforcer",
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
			Infrastructure: csp.Parameters.KhulnasoftCsp.Spec.Infrastructure,
			Common:         csp.Parameters.KhulnasoftCsp.Spec.Common,
			Gateway: &v1alpha1.KhulnasoftGatewayInformation{
				Host: fmt.Sprintf("%s-gateway", cr.Name),
				Port: 8443,
			},
			Secret: &v1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf("%s-enforcer-token", cr.Name),
				Key:  "token",
			},
			EnforcerService: &v1alpha1.KhulnasoftService{
				ImageData: &v1alpha1.KhulnasoftImage{
					Registry: registry,
				},
			},
			RunAsNonRoot:           csp.Parameters.KhulnasoftCsp.Spec.RunAsNonRoot,
			EnforcerUpdateApproved: csp.Parameters.KhulnasoftCsp.Spec.EnforcerUpdateApproved,
		},
	}

	return khulnasoftenf
}

func (csp *KhulnasoftCspHelper) newKhulnasoftKubeEnforcer(cr *v1alpha1.KhulnasoftCsp) *v1alpha1.KhulnasoftKubeEnforcer {
	registry := consts.Registry
	if cr.Spec.RegistryData != nil {
		if len(cr.Spec.RegistryData.URL) > 0 {
			registry = cr.Spec.RegistryData.URL
		}
	}
	if cr.Spec.DeployKubeEnforcer.Registry != "" {
		registry = cr.Spec.DeployKubeEnforcer.Registry
	}

	tag := consts.LatestVersion
	if cr.Spec.Infrastructure.Version != "" {
		tag = cr.Spec.Infrastructure.Version
	}
	if cr.Spec.DeployKubeEnforcer.ImageTag != "" {
		tag = cr.Spec.DeployKubeEnforcer.ImageTag
	}

	labels := map[string]string{
		"app":                cr.Name + "-csp",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
		"khulnasoft.component":     "kubeenforcer",
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
				GatewayAddress:  fmt.Sprintf("%s.%s:8443", fmt.Sprintf(consts.GatewayServiceName, cr.Name), cr.Namespace),
				ClusterName:     "Default-cluster-name",
				ImagePullSecret: cr.Spec.Common.ImagePullSecret,
			},
			Token:                  consts.DefaultKubeEnforcerToken,
			EnforcerUpdateApproved: cr.Spec.EnforcerUpdateApproved,
			AllowAnyVersion:        cr.Spec.Common.AllowAnyVersion,
			ImageData: &v1alpha1.KhulnasoftImage{
				Registry:   registry,
				Repository: "kube-enforcer",
				Tag:        tag,
				PullPolicy: "Always",
			},
			DeployStarboard: &KhulnasoftStarboardDetails,
		},
	}

	return khulnasoftKubeEnf
}

/*func (csp *KhulnasoftCspHelper) newKhulnasoftScanner(cr *operatorv1alpha1.KhulnasoftCsp) *operatorv1alpha1.KhulnasoftScanner {
	labels := map[string]string{
		"app":                cr.Name + "-csp",
		"deployedby":         "khulnasoft-operator",
		"khulnasoftoperator_cr": cr.Name,
	}
	annotations := map[string]string{
		"description": "Deploy Khulnasoft Scanner",
	}
	scanner := &operatorv1alpha1.KhulnasoftScanner{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.khulnasoft.com/v1alpha1",
			Kind:       "KhulnasoftScanner",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: operatorv1alpha1.KhulnasoftScannerSpec{
			Infrastructure: cr.Spec.Infrastructure,
			Common:         cr.Spec.Common,
			ScannerService: cr.Spec.ScannerService,
			Login: &operatorv1alpha1.KhulnasoftLogin{
				Username: "administrator",
				Password: cr.Spec.AdminPassword,
				Host:     fmt.Sprintf("http://%s:8080", fmt.Sprintf(consts.ServerServiceName, cr.Name)),
			},
		},
	}

	return scanner
}*/
