package consts

const (
	// ServiceAccount Service Account
	ServiceAccount = "%s-sa"

	// StarboardServiceAccount Service Account
	StarboardServiceAccount = "starboard-operator"
	// Registry URL
	Registry = "registry.khulnasoftsec.com"

	// StarboardRegistry URL
	StarboardRegistry = "docker.io/khulnasoftsec"

	// PullPolicy Image Pull Policy
	PullPolicy = "IfNotPresent"

	// PullImageSecretName pull image secret name
	PullImageSecretName = "%s-registry-secret"

	// AdminPasswordSecretName Admin Password Secre Name
	AdminPasswordSecretName = "%s-khulnasoft-admin"

	// AdminPasswordSecretKey Admin Password Secret Key
	AdminPasswordSecretKey = "password"

	// LicenseTokenSecretName License Token Secret Name
	LicenseTokenSecretName = "%s-khulnasoft-license"

	// LicenseTokenSecretKey License Token Secret Key
	LicenseTokenSecretKey = "license"

	// EnforcerTokenSecretName Enforcer Token Secret Name
	EnforcerTokenSecretName = "%s-enforcer-token"

	// EnforcerTokenSecretKey Enforcer Token Secret Key
	EnforcerTokenSecretKey = "token"

	// ScalockDbPasswordSecretName Scalock DB Password Secret Name
	ScalockDbPasswordSecretName = "%s-khulnasoft-db"

	// AuditDbPasswordSecretName Scalock audit DB Password Secret Name
	AuditDbPasswordSecretName = "%s-khulnasoft-audit-db"

	// ClusterReaderRole is Openshift cluster role to bind Khulnasoft service accounts
	ClusterReaderRole = "cluster-reader"

	// KhulnasoftSAClusterReaderRoleBind is Openshift cluster role binding between khulnasoft-sa and ClusterReaderRole
	KhulnasoftSAClusterReaderRoleBind = "khulnasoft-sa-cluster-reader-crb"

	// KhulnasoftKubeEnforcerSAClusterReaderRoleBind is Openshift cluster role binding between khulnasoft-kube-enforcer-sa and ClusterReaderRole
	KhulnasoftKubeEnforcerSAClusterReaderRoleBind = "khulnasoft-kube-enforcer-sa-cluster-reader-crb"

	KhulnasoftKubeEnforcerFinalizer                          = "khulnasoftkubeenforcers.operator.khulnasoftsec.com/finalizer"
	KhulnasoftKubeEnforcerMutantingWebhookConfigurationName  = "kube-enforcer-me-injection-hook-config"
	KhulnasoftKubeEnforcerValidatingWebhookConfigurationName = "kube-enforcer-admission-hook-config"
	KhulnasoftKubeEnforcerClusterRoleName                    = "khulnasoft-kube-enforcer"
	KhulnasoftKubeEnforcerClusterRoleBidingName              = "khulnasoft-kube-enforcer"

	// KhulnasoftStarboardSAClusterReaderRoleBind is Openshift cluster role binding between khulnasoft-starboard-sa and ClusterReaderRole
	KhulnasoftStarboardSAClusterReaderRoleBind = "khulnasoft-starboard-sa-cluster-reader-crb"

	// ScalockDbPasswordSecretKey Scalock DB Password Secret Key
	ScalockDbPasswordSecretKey = "password"

	// GatewayURL Khulnasoft Gateway
	GatewayURL = "%s-gateway:8443"

	// DiscoveryClusterRole Discovery Cluster Role
	DiscoveryClusterRole = "%s-discovery-cr"

	// DiscoveryClusterRoleBinding Discovery Cluster Role Binding
	DiscoveryClusterRoleBinding = "%s-discovery-crb"

	// DbPvcName DB PVC Name
	DbPvcName = "%s-db-pvc"

	// AuditDbPvcName DB PVC Name
	AuditDbPvcName = "%s-audit-db-pvc"

	// DbPvcSize Database PVC Size
	DbPvcSize = 10

	// LatestVersion Latest supported khulnasoft version in operator
	LatestVersion = "2022.4"

	// StarboardVersion Latest starboard version
	StarboardVersion = "0.15.10"

	// CyberCenterAddress Khulnasoft Cybercenter Address
	CyberCenterAddress = "https://cybercenter5.khulnasoftsec.com"

	// Deployments

	DbDeployName  = "%s-db"
	DbServiceName = "%s-db"

	AuditDbDeployName  = "%s-audit-db"
	AuditDbServiceName = "%s-audit-db"

	GatewayDeployName  = "%s-gateway"
	GatewayServiceName = "%s-gateway"

	ServerDeployName  = "%s-server"
	ServerServiceName = "%s-server"

	EnforcerDeamonsetName = "%s-agent"

	ScannerDeployName = "%s-scanner"

	ScannerSecretName = "khulnasoft-scanner"

	ScannerConfigMapName = "khulnasoft-scanner-config"

	EmptyString = ""

	KhulnasoftRunAsUser  = int64(11431)
	KhulnasoftRunAsGroup = int64(11433)
	KhulnasoftFsGroup    = int64(11433)

	DefaultKubeEnforcerToken = "ke-token"

	DBInitContainerCommand = "[ -f $PGDATA/server.key ] && chmod 600 $PGDATA/server.key || echo 'OK'"

	OpenShiftPlatform = "openshift"

	// mtls

	MtlsKhulnasoftWebSecretName = "khulnasoft-grpc-web"

	MtlsKhulnasoftGatewaySecretName = "khulnasoft-grpc-gateway"

	MtlsKhulnasoftEnforcerSecretName = "khulnasoft-grpc-enforcer"

	MtlsKhulnasoftKubeEnforcerSecretName = "khulnasoft-grpc-kube-enforcer"

	OperatorLogDevMode = "false"

	OperatorConcurrentScanJobsLimit = "10"

	OperatorScanJobRetryAfter = "30s"

	OperatorMetricsBindAddress = ":8080"

	OperatorHealthProbeBindAddress = ":9090"

	OperatorCisKubernetesBenchmarkEnabled = "false"

	OperatorVulnerabilityScannerEnabled = "false"

	OperatorBatchDeleteLimit = "10"

	OperatorBatchDeleteDelay = "10s"

	OperatorClusterComplianceEnabled = "false"

	ServerConfigMapName = "khulnasoft-csp-server-config"

	EnforcerConfigMapName = "khulnasoft-csp-enforcer"
)
