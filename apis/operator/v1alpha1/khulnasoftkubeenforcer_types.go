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

// KhulnasoftKubeEnforcerSpec defines the desired state of KhulnasoftKubeEnforcer
type KhulnasoftKubeEnforcerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Infrastructure         *KhulnasoftInfrastructure    `json:"infra,omitempty"`
	Config                 KhulnasoftKubeEnforcerConfig `json:"config"`
	Token                  string                 `json:"token,omitempty"`
	RegistryData           *KhulnasoftDockerRegistry    `json:"registry,omitempty"`
	ImageData              *KhulnasoftImage             `json:"image,omitempty"`
	EnforcerUpdateApproved *bool                  `json:"updateEnforcer,omitempty"`
	AllowAnyVersion        bool                   `json:"allowAnyVersion,omitempty"`
	KubeEnforcerService    *KhulnasoftService           `json:"deploy,omitempty"`
	Envs                   []corev1.EnvVar        `json:"env,omitempty"`
	Mtls                   bool                   `json:"mtls,omitempty"`
	DeployStarboard        *KhulnasoftStarboardDetails  `json:"starboard,omitempty"`
	ConfigMapChecksum      string                 `json:"config_map_checksum,omitempty"`
}

// KhulnasoftKubeEnforcerStatus defines the observed state of KhulnasoftKubeEnforcer
type KhulnasoftKubeEnforcerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft KubeEnforcer Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft KubeEnforcer status"

// KhulnasoftKubeEnforcer is the Schema for the khulnasoftkubeenforcers API
type KhulnasoftKubeEnforcer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftKubeEnforcerSpec   `json:"spec,omitempty"`
	Status KhulnasoftKubeEnforcerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftKubeEnforcerList contains a list of KhulnasoftKubeEnforcer
type KhulnasoftKubeEnforcerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftKubeEnforcer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftKubeEnforcer{}, &KhulnasoftKubeEnforcerList{})
}
