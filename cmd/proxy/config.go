package main

import (
	"log"
	"os"
)

func rootKey() []byte {
	if k := os.Getenv("MACAROON_KEY"); k != "" {
		return []byte(k)
	}
	return []byte("dev-insecure-key")
}

func policyPath() string {
	if p := os.Getenv("POLICY_FILE"); p != "" {
		return p
	}
	return "policies/agents.cedar"
}

func mustReadPolicy() []byte {
	b, err := os.ReadFile(policyPath())
	if err != nil {
		log.Fatal(err)
	}
	return b
}
