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

// KhulnasoftEnforcerSpec defines the desired state of KhulnasoftEnforcer
type KhulnasoftEnforcerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Infrastructure *KhulnasoftInfrastructure `json:"infra"`
	Common         *KhulnasoftCommon         `json:"common"`

	EnforcerService        *KhulnasoftService            `json:"deploy,required"`
	Gateway                *KhulnasoftGatewayInformation `json:"gateway,required"`
	Token                  string                  `json:"token,required"`
	Secret                 *KhulnasoftSecret             `json:"secret,omitempty"`
	Envs                   []corev1.EnvVar         `json:"env,omitempty"`
	RunAsNonRoot           bool                    `json:"runAsNonRoot,omitempty"`
	EnforcerUpdateApproved *bool                   `json:"updateEnforcer,omitempty"`
	Mtls                   bool                    `json:"mtls,omitempty"`
	ConfigMapChecksum      string                  `json:"config_map_checksum,omitempty"`
	KhulnasoftExpressMode        bool                    `json:"khulnasoft_express_mode,omitempty"`
	RhcosVersion           string                  `json:"rhcosVersion,omitempty"`
}

// KhulnasoftEnforcerStatus defines the observed state of KhulnasoftEnforcer
type KhulnasoftEnforcerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.deploy.replicas",description="Replicas Number"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft Enforcer Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft Enforcer status"

// KhulnasoftEnforcer is the Schema for the khulnasoftenforcers API
type KhulnasoftEnforcer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftEnforcerSpec   `json:"spec,omitempty"`
	Status KhulnasoftEnforcerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftEnforcerList contains a list of KhulnasoftEnforcer
type KhulnasoftEnforcerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftEnforcer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftEnforcer{}, &KhulnasoftEnforcerList{})
}
