package common

import (
	"fmt"

	"github.com/khulnasoft/khulnasoft-operator/controllers/ocp"
	"github.com/khulnasoft/khulnasoft-operator/pkg/utils/extra"

	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	"github.com/khulnasoft/khulnasoft-operator/pkg/consts"
)

func UpdateKhulnasoftInfrastructure(infra *operatorv1alpha1.KhulnasoftInfrastructure, name, namespace string) *operatorv1alpha1.KhulnasoftInfrastructure {
	return UpdateKhulnasoftInfrastructureFull(infra, name, namespace, "")
}

func UpdateKhulnasoftInfrastructureFull(infra *operatorv1alpha1.KhulnasoftInfrastructure, name, namespace, image string) *operatorv1alpha1.KhulnasoftInfrastructure {
	if infra != nil {
		if len(infra.Namespace) == 0 {
			infra.Namespace = namespace
		}

		if len(infra.ServiceAccount) == 0 {
			if image == "starboard" {
				infra.ServiceAccount = consts.StarboardServiceAccount
			} else {
				infra.ServiceAccount = fmt.Sprintf(consts.ServiceAccount, name)
			}

		}

		if len(infra.Version) == 0 {
			if image == "starboard" {
				infra.Version = consts.StarboardVersion
			} else {
				infra.Version = consts.LatestVersion
			}
		}

		if len(infra.Platform) == 0 {
			isOpenshift, _ := ocp.VerifyRouteAPI()
			if isOpenshift {
				infra.Platform = "openshift"
			} else {
				infra.Platform = "kubernetes"
			}
		}
	} else {
		infra = &operatorv1alpha1.KhulnasoftInfrastructure{
			ServiceAccount: fmt.Sprintf(consts.ServiceAccount, name),
			Namespace:      namespace,
			Version:        consts.LatestVersion,
			Platform:       "openshift",
			Requirements:   false,
		}
	}

	return infra
}

func UpdateKhulnasoftCommon(common *operatorv1alpha1.KhulnasoftCommon, name string, admin bool, license bool) *operatorv1alpha1.KhulnasoftCommon {
	if common != nil {
		if len(common.CyberCenterAddress) == 0 {
			common.CyberCenterAddress = consts.CyberCenterAddress
		}

		if len(common.ImagePullSecret) == 0 {
			marketplace := extra.IsMarketPlace()
			if !marketplace {
				common.ImagePullSecret = fmt.Sprintf(consts.PullImageSecretName, name)
			}
		}

		if common.AdminPassword == nil && admin {
			common.AdminPassword = &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.AdminPasswordSecretName, name),
				Key:  consts.AdminPasswordSecretKey,
			}
		}

		if common.KhulnasoftLicense == nil && license {
			common.KhulnasoftLicense = &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.LicenseTokenSecretName, name),
				Key:  consts.LicenseTokenSecretKey,
			}
		}

		if common.DatabaseSecret == nil {
			common.DatabaseSecret = &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.ScalockDbPasswordSecretName, name),
				Key:  consts.ScalockDbPasswordSecretKey,
			}
		}

		if common.DbDiskSize == 0 {
			common.DbDiskSize = consts.DbPvcSize
		}
	} else {
		adminPassword := (*operatorv1alpha1.KhulnasoftSecret)(nil)
		khulnasoftLicense := (*operatorv1alpha1.KhulnasoftSecret)(nil)

		if admin {
			adminPassword = &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.AdminPasswordSecretName, name),
				Key:  consts.AdminPasswordSecretKey,
			}
		}

		if license {
			khulnasoftLicense = &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.LicenseTokenSecretName, name),
				Key:  consts.LicenseTokenSecretKey,
			}
		}

		common = &operatorv1alpha1.KhulnasoftCommon{
			ActiveActive:       false,
			CyberCenterAddress: consts.CyberCenterAddress,
			ImagePullSecret:    fmt.Sprintf(consts.PullImageSecretName, name),
			AdminPassword:      adminPassword,
			KhulnasoftLicense:  khulnasoftLicense,
			DatabaseSecret: &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.ScalockDbPasswordSecretName, name),
				Key:  consts.ScalockDbPasswordSecretKey,
			},
			DbDiskSize:      consts.DbPvcSize,
			SplitDB:         false,
			AllowAnyVersion: false,
		}
	}

	return common
}

func UpdateKhulnasoftAuditDB(auditDb *operatorv1alpha1.AuditDBInformation, name string) *operatorv1alpha1.AuditDBInformation {
	password := extra.CreateRundomPassword()

	if auditDb != nil {
		if auditDb.AuditDBSecret == nil {
			auditDb.AuditDBSecret = &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.AuditDbPasswordSecretName, name),
				Key:  consts.ScalockDbPasswordSecretKey,
			}
		}

		if auditDb.Data == nil {
			auditDb.Data = &operatorv1alpha1.KhulnasoftDatabaseInformation{
				Host:     fmt.Sprintf(consts.AuditDbServiceName, name),
				Port:     5432,
				Username: "postgres",
				Password: password,
			}
		}
	} else {
		auditDb = &operatorv1alpha1.AuditDBInformation{
			AuditDBSecret: &operatorv1alpha1.KhulnasoftSecret{
				Name: fmt.Sprintf(consts.AuditDbPasswordSecretName, name),
				Key:  consts.ScalockDbPasswordSecretKey,
			},
			Data: &operatorv1alpha1.KhulnasoftDatabaseInformation{
				Host:     fmt.Sprintf(consts.AuditDbServiceName, name),
				Port:     5432,
				Username: "postgres",
				Password: password,
			},
		}
	}

	return auditDb
}
