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

// KhulnasoftServerSpec defines the desired state of KhulnasoftServer
type KhulnasoftServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Infrastructure *KhulnasoftInfrastructure `json:"infra"`
	Common         *KhulnasoftCommon         `json:"common"`

	ServerService     *KhulnasoftService             `json:"deploy,required"`
	ExternalDb        *KhulnasoftDatabaseInformation `json:"externalDb,omitempty"`
	AuditDB           *AuditDBInformation      `json:"auditDB,omitempty"`
	LicenseToken      string                   `json:"licenseToken,omitempty"`
	AdminPassword     string                   `json:"adminPassword,omitempty"`
	Enforcer          *KhulnasoftEnforcerDetailes    `json:"enforcer,omitempty"`
	Envs              []corev1.EnvVar          `json:"env,omitempty"`
	ConfigMapData     map[string]string        `json:"configMapData,omitempty"`
	RunAsNonRoot      bool                     `json:"runAsNonRoot,omitempty"`
	Route             bool                     `json:"route,omitempty"`
	Mtls              bool                     `json:"mtls,omitempty"`
	ConfigMapChecksum string                   `json:"config_map_checksum,omitempty"`
}

// KhulnasoftServerStatus defines the observed state of KhulnasoftServer
type KhulnasoftServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Nodes []string            `json:"nodes"`
	State KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.deploy.replicas",description="Replicas Number"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft Server Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft Server status"
//+kubebuilder:printcolumn:name="Nodes",type="string",JSONPath=".status.nodes",description="List Of Nodes (Pods)"

// KhulnasoftServer is the Schema for the khulnasoftservers API
type KhulnasoftServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftServerSpec   `json:"spec,omitempty"`
	Status KhulnasoftServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftServerList contains a list of KhulnasoftServer
type KhulnasoftServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftServer{}, &KhulnasoftServerList{})
}
