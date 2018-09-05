package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"

	appdbv1 "github.com/danisla/appdb-operator/pkg/types"
)

func myLog(parent *appdbv1.AppDBInstance, level, msg string) {
	log.Printf("[%s][%s][%s] %s", level, parent.Kind, parent.Name, msg)
}

func calcParentSig(spec interface{}, addStr string) string {
	hasher := sha1.New()
	data, err := json.Marshal(&spec)
	if err != nil {
		log.Printf("[ERROR] Failed to convert parent spec to JSON, this is a bug.\n")
		return ""
	}
	hasher.Write([]byte(data))
	hasher.Write([]byte(addStr))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
