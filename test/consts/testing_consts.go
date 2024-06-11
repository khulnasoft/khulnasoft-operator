package testing_consts

import "github.com/khulnasoft/khulnasoft-operator/pkg/consts"

const (
	Namespace                  = "khulnasoft"
	Version                    = consts.LatestVersion
	StarboardVersion           = consts.StarboardVersion
	UbiImageTag                = "2022.4.81-ubi8" //todo: update it to latest after we create ubi8 latest tag in the release process
	NameSpace                  = "khulnasoft"
	CspServiceAccount          = "khulnasoft-sa"
	StarboardServiceAccount    = "starboard-operator"
	KubeEnforcerServiceAccount = "khulnasoft-kube-enforcer-sa"
	ImagePullSecret            = "khulnasoft-registry"
	StarboardImagePullSecret   = "starboard-registry"
	DbDiskSize                 = 10
	DataBaseSecretKey          = "db-password"
	DatabaseSecretName         = "khulnasoft-database-password"
	Registry                   = "registry.khulnasoft.com"
	StarboardRegistry          = "docker.io/khulnasoft"
	DatabaseRepo               = "database"
	GatewayRepo                = "gateway"
	ServerRepo                 = "console"
	EnforcerRepo               = "enforcer"
	KeEnforcerRepo             = "kube-enforcer"
	ScannerRepo                = "scanner"
	StarboardRepo              = "starboard-operator"
	GatewayPort                = 8443
	DbPvcStorageClassName      = "khulnasoft-storage"
	DbPvcStorageSize           = "50Gi"
	DbPvcHostPath              = "/tmp/khulnasoftdb/"
	EnforcerToken              = "enforcer_token"
	GatewayServiceName         = "%s-gateway"
	EnforcerGroupName          = "operator-default-enforcer-group"
	KubeEnforcerToken          = "ke_enforcer_token"
	KUbeEnforcerGroupName      = "operator-default-ke-group"
	ServerAdminUser            = "administrator"
	ServerAdminPassword        = "@Password1"
	ServerHost                 = "http://khulnasoft-server:8080"
	ScannerToken               = ""
	GatewayAddress             = "khulnasoft-gateway:8443"
	ClusterName                = "Default-cluster-name"
)
