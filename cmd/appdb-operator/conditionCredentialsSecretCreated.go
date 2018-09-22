package main

import (
	"fmt"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	tfv1 "github.com/danisla/terraform-operator/pkg/types"
)

func reconcileSecretCreated(condition *appdbv1.AppDBCondition, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}, appdbi appdbv1.AppDBInstance, tfapply tfv1.Terraform) appdbv1.ConditionStatus {
	newStatus := appdbv1.ConditionFalse

	// Generate secret for DB credentials.
	newStatus = appdbv1.ConditionFalse
	if passwordsVar, ok := tfapply.Status.TFOutput["user_passwords"]; ok == true {
		passwords := strings.Split(passwordsVar.Value, ",")
		if len(parent.Spec.Users) != len(passwords) {
			condition.Reason = fmt.Sprintf("passwords output from TerraformApply is different length than input users.")
		} else {
			status.CredentialsSecrets = make(map[string]string, 0)
			secretNames := []string{}
			for i := 0; i < len(parent.Spec.Users); i++ {
				secretName := fmt.Sprintf("appdb-%s-%s-user-%d", appdbi.GetName(), parent.GetName(), i)

				secret := makeCredentialsSecret(secretName, parent.GetNamespace(), parent.Spec.Users[i], passwords[i], parent.Spec.DBName, appdbi.Status.DBHost, appdbi.Status.DBPort)

				secretNames = append(secretNames, secretName)

				status.CredentialsSecrets[parent.Spec.Users[i]] = secretName

				claimChildAndGetCurrent(secret, children, desiredChildren)

				newStatus = appdbv1.ConditionTrue
			}
			condition.Reason = fmt.Sprintf("Secret/%s: CREATED", strings.Join(secretNames, ","))
		}
	} else {
		condition.Reason = "No user_passwords found in output varibles of TerraformApply status"
	}

	return newStatus
}
