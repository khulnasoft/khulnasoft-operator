package common

import (
	"errors"
	operatorv1alpha1 "github.com/khulnasoft/khulnasoft-operator/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type EnvsParameters struct {
	Infrastructure *operatorv1alpha1.KhulnasoftInfrastructure
	Common         *operatorv1alpha1.KhulnasoftCommon
	ExternalDb     *operatorv1alpha1.KhulnasoftDatabaseInformation
	Name           string
	AuditDB        *operatorv1alpha1.AuditDBInformation
}

type KhulnasoftEnvsHelper struct {
	Parameters EnvsParameters
}

func NewKhulnasoftEnvsHelper(infra *operatorv1alpha1.KhulnasoftInfrastructure,
	common *operatorv1alpha1.KhulnasoftCommon,
	externalDb *operatorv1alpha1.KhulnasoftDatabaseInformation,
	name string,
	auditDB *operatorv1alpha1.AuditDBInformation) *KhulnasoftEnvsHelper {
	params := EnvsParameters{
		Infrastructure: infra,
		Common:         common,
		ExternalDb:     externalDb,
		Name:           name,
		AuditDB:        auditDB,
	}

	return &KhulnasoftEnvsHelper{
		Parameters: params,
	}
}

func (ctx *KhulnasoftEnvsHelper) GetDbEnvVars() ([]corev1.EnvVar, error) {
	result := ([]corev1.EnvVar)(nil)

	dbSecret := ctx.Parameters.Common.DatabaseSecret
	dbAuditSecret := dbSecret

	if ctx.Parameters.Common.SplitDB {
		dbAuditSecret = ctx.Parameters.AuditDB.AuditDBSecret
	}

	result = []corev1.EnvVar{
		{
			Name: "SCALOCK_DBPASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dbSecret.Name,
					},
					Key: dbSecret.Key,
				},
			},
		},
		{
			Name: "SCALOCK_AUDIT_DBPASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dbAuditSecret.Name,
					},
					Key: dbAuditSecret.Key,
				},
			},
		},
	}

	if ctx.Parameters.Common.ActiveActive {
		item := corev1.EnvVar{
			Name: "KHULNASOFT_PUBSUB_DBPASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dbSecret.Name,
					},
					Key: dbSecret.Key,
				},
			},
		}
		result = append(result, item)
	}

	if result == nil {
		return nil, errors.New("Failed to create Khulnasoft Gateway deployments environments variables")
	}

	return result, nil
}
