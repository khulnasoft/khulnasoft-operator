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
	"github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KhulnasoftStarboardSpec defines the desired state of KhulnasoftStarboard
type KhulnasoftStarboardSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Infrastructure                   *v1alpha1.KhulnasoftInfrastructure `json:"infra,omitempty"`
	AllowAnyVersion                  bool                         `json:"allowAnyVersion,omitempty"`
	StarboardService                 *v1alpha1.KhulnasoftService        `json:"deploy,required"`
	Config                           v1alpha1.KhulnasoftStarboardConfig `json:"config"`
	RegistryData                     *v1alpha1.KhulnasoftDockerRegistry `json:"registry,omitempty"`
	ImageData                        *v1alpha1.KhulnasoftImage          `json:"image,omitempty"`
	Envs                             []corev1.EnvVar              `json:"env,omitempty"`
	KubeEnforcerVersion              string                       `json:"kube_enforcer_version,omitempty"`
	LogDevMode                       bool                         `json:"logDevMode,omitempty"`
	ConcurrentScanJobsLimit          string                       `json:"concurrentScanJobsLimit,omitempty"`
	ScanJobRetryAfter                string                       `json:"scanJobRetryAfter,omitempty"`
	MetricsBindAddress               string                       `json:"metricsBindAddress,omitempty"`
	HealthProbeBindAddress           string                       `json:"healthProbeBindAddress,omitempty"`
	CisKubernetesBenchmarkEnabled    string                       `json:"cisKubernetesBenchmarkEnabled,omitempty"`
	VulnerabilityScannerEnabled      string                       `json:"vulnerabilityScannerEnabled,omitempty"`
	BatchDeleteLimit                 string                       `json:"batchDeleteLimit,omitempty"`
	BatchDeleteDelay                 string                       `json:"batchDeleteDelay,omitempty"`
	OperatorClusterComplianceEnabled string                       `json:"operator_cluster_compliance_enabled"`
	ConfigMapChecksum                string                       `json:"config_map_checksum,omitempty"`
}

// KhulnasoftStarboardStatus defines the observed state of KhulnasoftStarboard
type KhulnasoftStarboardStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Nodes []string                     `json:"nodes"`
	State v1alpha1.KhulnasoftDeploymentState `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.deploy.replicas",description="Replicas Number"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath="..metadata.creationTimestamp",description="Khulnasoft Starboard Age"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="Khulnasoft Starboard status"
//+kubebuilder:printcolumn:name="Nodes",type="string",JSONPath=".status.nodes",description="List Of Nodes (Pods)"

// KhulnasoftStarboard is the Schema for the khulnasoftstarboards API
type KhulnasoftStarboard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KhulnasoftStarboardSpec   `json:"spec,omitempty"`
	Status KhulnasoftStarboardStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KhulnasoftStarboardList contains a list of KhulnasoftStarboard
type KhulnasoftStarboardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KhulnasoftStarboard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KhulnasoftStarboard{}, &KhulnasoftStarboardList{})
}
