package model

import (
	"errors"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

const guardianDeploymentTermsKey = "GuardianDeploymentAfterSalesTerms"
const guardianMacTermsKey = "GuardianMacAfterSalesTerms"

type GuardianAfterSalesTerms struct {
	Deployment string `json:"deployment"`
	Mac        string `json:"mac"`
}

func GetGuardianAfterSalesTerms(tx *gorm.DB) (GuardianAfterSalesTerms, error) {
	var options []Option
	if err := tx.Where(map[string]any{"key": []string{guardianDeploymentTermsKey, guardianMacTermsKey}}).Find(&options).Error; err != nil {
		return GuardianAfterSalesTerms{}, err
	}
	terms := GuardianAfterSalesTerms{}
	for _, option := range options {
		switch option.Key {
		case guardianDeploymentTermsKey:
			terms.Deployment = option.Value
		case guardianMacTermsKey:
			terms.Mac = option.Value
		}
	}
	return terms, nil
}

func SaveGuardianAfterSalesTerms(terms GuardianAfterSalesTerms) error {
	terms.Deployment = strings.TrimSpace(terms.Deployment)
	terms.Mac = strings.TrimSpace(terms.Mac)
	if utf8.RuneCountInString(terms.Deployment) > 20000 || utf8.RuneCountInString(terms.Mac) > 20000 {
		return errors.New("售后条款每类不能超过 20000 字")
	}
	return UpdateOptionsBulk(map[string]string{guardianDeploymentTermsKey: terms.Deployment, guardianMacTermsKey: terms.Mac})
}
