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

// KhulnasoftLightningSpec defines the desired state of KhulnasoftLightning
type KhulnasoftLightningSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Common       *KhulnasoftCommon                `json:"common"`
	Global       *KhulnasoftLightningGlobal       `json:"global"`
	KubeEnforcer *KhulnasoftLightningKubeEnforcer `json:"kubeEnforcer"`
	Enforcer     *KhulnasoftLightningEnforcer     `json:"enforcer"`
}

type KhulnasoftLightningGlobal struct {
	GatewayAddress string `json:"gateway_address,omitempty"`
	ClusterName    string `json:"cluster_name,omitempty"`
}

type KhulnasoftLightningKubeEnforcer struct {
	Infrastructure         *KhulnasoftInfrastructure   `json:"infra,omitempty"`
	Token                  string                      `json:"token,omitempty"`
	RegistryData           *KhulnasoftDockerRegistry   `json:"registry,omitempty"`
	ImageData              *KhulnasoftImage            `json:"image,omitempty"`
	EnforcerUpdateApproved *bool                       `json:"updateEnforcer,omitempty"`
	AllowAnyVersion        bool                        `json:"allowAnyVersion,omitempty"`
	KubeEnforcerService    *KhulnasoftService          `json:"deploy,omitempty"`
	Envs                   []corev1.EnvVar             `json:"env,omitempty"`
	Mtls                   bool                        `json:"mtls,omitempty"`
	DeployStarboard        *KhulnasoftStarboardDetails `json:"starboard,omitempty"`
	ConfigMapChecksum      string                      `json:"config_map_checksum,omitempty"`
}

type KhulnasoftLightningEnforcer struct {
	Infrastructure         *KhulnasoftInfrastructure `json:"infra"`
	EnforcerService        *KhulnasoftService        `json:"deploy,required"`
	Token                  string                    `json:"token,omitempty"`
	Secret                 *KhulnasoftSecret         `json:"secret,omitempty"`
	Envs                   []corev1.EnvVar           `json:"env,omitempty"`
	RunAsNonRoot           bool                      `json:"runAsNonRoot,omitempty"`
	EnforcerUpdateApproved *bool                     `json:"updateEnforcer,omitempty"`
	Mtls                   bool                      `json:"mtls,omitempty"`
	ConfigMapChecksum      string                    `json:"config_map_checksum,omitempty"`
	KhulnasoftExpressMode  bool                      `json:"khulnasoft_express_mode,omitempty"`
	RhcosVersion           string                    `json:"rhcosVersion,omitempty"`
}

// KhulnasoftLightningStatus defines the observed state of KhulnasoftLightning
type KhulnasoftLightningStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft Lightning Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft Lightning status"

// KhulnasoftLightning is the Schema for the khulnasoftLightnings API
type KhulnasoftLightning struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftLightningSpec   `json:"spec,omitempty"`
	Status KhulnasoftLightningStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftLightningList contains a list of KhulnasoftLightning
type KhulnasoftLightningList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftLightning `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftLightning{}, &KhulnasoftLightningList{})
}
