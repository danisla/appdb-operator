package main

import (
	"fmt"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func verifySpec(parent *appdbv1.AppDB) error {
	if parent.Spec.AppDBInstance == "" {
		return fmt.Errorf("Missing spec.appDBInstance")
	}

	if parent.Spec.DBName == "" {
		return fmt.Errorf("Missing spec.dbName")
	}

	if len(parent.Spec.Users) == 0 {
		return fmt.Errorf("spec.users list is empty, must have at least 1 user")
	}

	return nil
}
