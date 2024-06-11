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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KhulnasoftCloudConnectorSpec defines the desired state of KhulnasoftCloudConnector
type KhulnasoftCloudConnectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Infrastructure *KhulnasoftInfrastructure `json:"infra"`
	Common         *KhulnasoftCommon         `json:"common"`

	CloudConnectorService *KhulnasoftService                 `json:"deploy,required"`
	Login                 *KhulnasoftLogin                   `json:"login,required"`
	RunAsNonRoot          bool                               `json:"runAsNonRoot,omitempty"`
	Tunnels               []*KhulnasoftCloudConnectorTunnels `json:"tunnels,required"`
	ConfigMapChecksum     string                             `json:"config_map_checksum,omitempty"`
}

// KhulnasoftCloudConnectorStatus defines the observed state of KhulnasoftCloudConnector
type KhulnasoftCloudConnectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Nodes []string                  `json:"nodes"`
	State KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.deploy.replicas",description="Replicas Number"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft Cloud Connector Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft Cloud Connector status"
//+kubebuilder:printcolumn:name="Nodes",type="string",JSONPath=".status.nodes",description="List Of Nodes (Pods)"

// KhulnasoftCloudConnector is the Schema for the khulnasoftcloudconnectors API
type KhulnasoftCloudConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftCloudConnectorSpec   `json:"spec,omitempty"`
	Status KhulnasoftCloudConnectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftCloudConnectorList contains a list of KhulnasoftCloudConnector
type KhulnasoftCloudConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftCloudConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftCloudConnector{}, &KhulnasoftCloudConnectorList{})
}
