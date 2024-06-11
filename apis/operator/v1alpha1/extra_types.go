package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type KhulnasoftInfrastructure struct {
	ServiceAccount string `json:"serviceAccount,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	Version        string `json:"version,omitempty"`
	Platform       string `json:"platform,omitempty"`
	Requirements   bool   `json:"requirements"`
}

type KhulnasoftCommon struct {
	ActiveActive       bool        `json:"activeActive"`
	StorageClass       string      `json:"storageclass,omitempty"`
	CyberCenterAddress string      `json:"cybercenterAddress,omitempty"`
	ImagePullSecret    string      `json:"imagePullSecret,omitempty"`
	AdminPassword      *KhulnasoftSecret `json:"adminPassword,omitempty"`
	KhulnasoftLicense        *KhulnasoftSecret `json:"license,omitempty"`
	DatabaseSecret     *KhulnasoftSecret `json:"databaseSecret,omitempty"`
	DbDiskSize         int         `json:"dbDiskSize,omitempty"`
	SplitDB            bool        `json:"splitDB,omitempty"`
	AllowAnyVersion    bool        `json:"allowAnyVersion,omitempty"`
}

type KhulnasoftDockerRegistry struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type KhulnasoftDatabaseInformation struct {
	Host     string `json:"host"`
	Port     int64  `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type KhulnasoftSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type KhulnasoftImage struct {
	Repository string `json:"repository"`
	Registry   string `json:"registry"`
	Tag        string `json:"tag"`
	PullPolicy string `json:"pullPolicy"`
}

// KhulnasoftService Struct for deployment spec
type KhulnasoftService struct {
	// Number of instances to deploy for a specific khulnasoft deployment.
	Replicas       int64                        `json:"replicas"`
	ServiceType    string                       `json:"service,omitempty"`
	ImageData      *KhulnasoftImage                   `json:"image,omitempty"`
	Resources      *corev1.ResourceRequirements `json:"resources,omitempty"`
	LivenessProbe  *corev1.Probe                `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe                `json:"readinessProbe,omitempty"`
	NodeSelector   map[string]string            `json:"nodeSelector,omitempty"`
	Affinity       *corev1.Affinity             `json:"affinity,omitempty"`
	Tolerations    []corev1.Toleration          `json:"tolerations,omitempty"`
	VolumeMounts   []corev1.VolumeMount         `json:"volumeMounts,omitempty"`
	Volumes        []corev1.Volume              `json:"volumes,omitempty"`
}

type KhulnasoftGatewayInformation struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
}

type KhulnasoftLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Token    string `json:"token"`
	Insecure bool   `json:"tlsNoVerify"`
}

type KhulnasoftScannerCliScale struct {
	Max              int64 `json:"max"`
	Min              int64 `json:"min"`
	ImagesPerScanner int64 `json:"imagesPerScanner"`
}

type KhulnasoftEnforcerDetailes struct {
	Gateway     string `json:"gateway"`
	Name        string `json:"name"`
	EnforceMode bool   `json:"enforceMode"`
}

type KhulnasoftDeploymentState string

const (
	// KhulnasoftDeploymentStatePending Pending status when start to deploy khulnasoft
	KhulnasoftDeploymentStatePending KhulnasoftDeploymentState = "Pending"

	// KhulnasoftDeploymentStateWaitingDB After creating khulnasoft database waiting to done
	KhulnasoftDeploymentStateWaitingDB KhulnasoftDeploymentState = "Waiting For Khulnasoft Database"

	// KhulnasoftDeploymentStateWaitingKhulnasoft After creating khulnasoft server and gateway waiting to done
	KhulnasoftDeploymentStateWaitingKhulnasoft KhulnasoftDeploymentState = "Waiting For Khulnasoft Server and Gateway"

	// KhulnasoftDeploymentStateRunning done
	KhulnasoftDeploymentStateRunning KhulnasoftDeploymentState = "Running"

	// KhulnasoftEnforcerUpdatePendingApproval Waiting for approval to update enforcer
	KhulnasoftEnforcerUpdatePendingApproval KhulnasoftDeploymentState = "Pending Approval for Enforcers Update"

	// KhulnasoftDeploymentUpdateInProgress When Operand is Updating to latest changes
	KhulnasoftDeploymentUpdateInProgress KhulnasoftDeploymentState = "Update In Progress"

	// KhulnasoftEnforcerUpdateInProgress When Enforcers Updating to latest changes
	KhulnasoftEnforcerUpdateInProgress KhulnasoftDeploymentState = "Enforcers Update In Progress"

	// KhulnasoftEnforcerWaiting Waiting for Enforcer inital Run
	KhulnasoftEnforcerWaiting KhulnasoftDeploymentState = "Waiting For Enforcers to Start"
)

type KhulnasoftKubeEnforcerConfig struct {
	GatewayAddress  string `json:"gateway_address,omitempty"`
	ClusterName     string `json:"cluster_name,omitempty"`
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

type KhulnasoftKubeEnforcerDetails struct {
	ImageTag string `json:"tag,omitempty"`
	Registry string `json:"registry,omitempty"`
}

type KhulnasoftStarboardConfig struct {
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

type KhulnasoftStarboardDetails struct {
	Infrastructure                *KhulnasoftInfrastructure `json:"infra,omitempty"`
	AllowAnyVersion               bool                `json:"allowAnyVersion,omitempty"`
	StarboardService              *KhulnasoftService        `json:"deploy,required"`
	Config                        KhulnasoftStarboardConfig `json:"config"`
	RegistryData                  *KhulnasoftDockerRegistry `json:"registry,omitempty"`
	ImageData                     *KhulnasoftImage          `json:"image,omitempty"`
	Envs                          []corev1.EnvVar     `json:"env,omitempty"`
	LogDevMode                    bool                `json:"logDevMode,omitempty"`
	ConcurrentScanJobsLimit       string              `json:"concurrentScanJobsLimit,omitempty"`
	ScanJobRetryAfter             string              `json:"scanJobRetryAfter,omitempty"`
	MetricsBindAddress            string              `json:"metricsBindAddress,omitempty"`
	HealthProbeBindAddress        string              `json:"healthProbeBindAddress,omitempty"`
	CisKubernetesBenchmarkEnabled string              `json:"cisKubernetesBenchmarkEnabled,omitempty"`
	VulnerabilityScannerEnabled   string              `json:"vulnerabilityScannerEnabled,omitempty"`
	BatchDeleteLimit              string              `json:"batchDeleteLimit,omitempty"`
	BatchDeleteDelay              string              `json:"batchDeleteDelay,omitempty"`
	ImageTag                      string              `json:"tag,omitempty"`
}

type AuditDBInformation struct {
	AuditDBSecret *KhulnasoftSecret              `json:"secret,omitempty"`
	Data          *KhulnasoftDatabaseInformation `json:"information,omitempty"`
}
