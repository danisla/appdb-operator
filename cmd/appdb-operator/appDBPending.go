package main

import (
	"fmt"
	"strings"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func stateCloudSQLDBRunning(parentType ParentType, parent *appdbv1.AppDB, status *appdbv1.AppDBOperatorStatus, children *AppDBChildren, desiredChildren *[]interface{}) (appdbv1.AppDBOperatorState, error) {
	status.Provisioning = appdbv1.ProvisioningStatusPending

	tfapply, ok := children.TerraformApplys[status.CloudSQLDB.TFApplyName]
	if ok == false {
		myLog(parent, "WARN", fmt.Sprintf("TerraformApply not found in children while in state %s", status.StateCurrent))
		return StateCloudSQLDBPending, nil
	}

	status.CloudSQLDB.TFApplyPodName = tfapply.Status.PodName

	switch tfapply.Status.PodStatus {
	case "FAILED":
		myLog(parent, "ERROR", "CloudSQLDB TerraformApply failed.")
		status.Provisioning = appdbv1.ProvisioningStatusFailed

		// Set the parent signature
		// If parent changes from here on, we'll go back through the idle state, creating new resources.
		status.LastAppliedSig = calcParentSig(parent, "")

		return StateWaitComplete, nil
	case "COMPLETED":
		myLog(parent, "INFO", "CloudSQLDB TerraformApply completed.")
		status.Provisioning = appdbv1.ProvisioningStatusComplete

		// Set the parent signature
		// If parent changes from here on, we'll go back through the idle state, creating new resources.
		status.LastAppliedSig = calcParentSig(parent, "")

		// Save credentials to new secret.
		if passwordsVar, ok := tfapply.Status.TFOutput["user_passwords"]; ok == true {
			secretName := status.CloudSQLDB.TFApplyName

			passwords := strings.Split(passwordsVar.Value, ",")
			if len(parent.Spec.Users) != len(passwords) {
				myLog(parent, "ERROR", "passwords output from TerraformApply is different length than input users.")
				return StateWaitComplete, nil
			}
			secret := makeCredentialsSecret(secretName, parent.ObjectMeta.Namespace, parent.Spec.Users, passwords)

			*desiredChildren = append(*desiredChildren, secret)

			myLog(parent, "INFO", fmt.Sprintf("Created credentials secret: %s", secretName))

			status.CredentialsSecret = secretName

		} else {
			myLog(parent, "ERROR", "No user_passwords found in output varibles of TerraformApply status")
			return StateWaitComplete, nil
		}

		return StateWaitComplete, nil
	}

	return StateCloudSQLDBPending, nil
}

func makeCredentialsSecret(name string, namespace string, users []string, passwords []string) corev1.Secret {
	var secret corev1.Secret

	data := make(map[string]string, 0)

	for i := 0; i < len(users); i++ {
		data[users[i]] = passwords[i]
	}

	secret = corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: data,
	}

	return secret
}
