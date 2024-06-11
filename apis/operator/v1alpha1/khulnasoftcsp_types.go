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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KhulnasoftCspSpec defines the desired state of KhulnasoftCsp
type KhulnasoftCspSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Infrastructure *KhulnasoftInfrastructure `json:"infra,omitempty"`
	Common         *KhulnasoftCommon         `json:"common,omitempty"`

	RegistryData *KhulnasoftDockerRegistry      `json:"registry,omitempty"`
	ExternalDb   *KhulnasoftDatabaseInformation `json:"externalDb,omitempty"`
	AuditDB      *AuditDBInformation      `json:"auditDB,omitempty"`

	DbService      *KhulnasoftService `json:"database,omitempty"`
	GatewayService *KhulnasoftService `json:"gateway,required"`
	ServerService  *KhulnasoftService `json:"server,required"`

	LicenseToken           string                   `json:"licenseToken,omitempty"`
	AdminPassword          string                   `json:"adminPassword,omitempty"`
	Enforcer               *KhulnasoftEnforcerDetailes    `json:"enforcer,omitempty"`
	Route                  bool                     `json:"route,omitempty"`
	RunAsNonRoot           bool                     `json:"runAsNonRoot,omitempty"`
	ServerEnvs             []corev1.EnvVar          `json:"serverEnvs,omitempty"`
	GatewayEnvs            []corev1.EnvVar          `json:"gatewayEnvs,omitempty"`
	ServerConfigMapData    map[string]string        `json:"serverConfigMapData,omitempty"`
	DeployKubeEnforcer     *KhulnasoftKubeEnforcerDetails `json:"kubeEnforcer,omitempty"`
	EnforcerUpdateApproved *bool                    `json:"updateEnforcer,omitempty"`
	Mtls                   bool                     `json:"mtls,omitempty"`
}

// KhulnasoftCspStatus defines the observed state of KhulnasoftCsp
type KhulnasoftCspStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase string              `json:"phase"`
	State KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft Csp Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft Csp status"

// KhulnasoftCsp is the Schema for the khulnasoftcsps API
type KhulnasoftCsp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftCspSpec   `json:"spec,omitempty"`
	Status KhulnasoftCspStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftCspList contains a list of KhulnasoftCsp
type KhulnasoftCspList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftCsp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftCsp{}, &KhulnasoftCspList{})
}
