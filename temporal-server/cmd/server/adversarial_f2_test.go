package main

import (
	"testing"

	"go.temporal.io/server/common/dynamicconfig"
)

// F2 item 3, the namespace half (TICK-122, F2-3b). The Authorizer gates a
// task-token call on the request's `namespace` field, but the frontend acts on
// the namespace id inside the unsigned token (v1.29.7 workflow_handler.go
// RespondActivityTaskCompleted). Only frontend.enableTokenNamespaceEnforcement
// (default true) refuses a token whose namespace differs from the request's.
// Turned off, a nava-worker certificate sending namespace "nava" with a token
// naming the temporal-system namespace id acts outside `nava`. D6-K must refuse
// to start with it off (ADR-001-A2, TICK-132).
func TestAdversarialF2TokenNamespaceEnforcementGuarded(t *testing.T) {
	if err := checkSecurityConfig(validConfig(t), shippedDC()); err != nil {
		t.Fatalf("shipped config refused: %v", err)
	}

	key := dynamicconfig.EnableTokenNamespaceEnforcement.Key()
	for name, v := range map[string]any{
		"false":        false,
		"string false": "false",
		"not a bool":   "maybe",
	} {
		t.Run(name, func(t *testing.T) {
			dc := shippedDC()
			dc[key] = v
			requireRefused(t, checkSecurityConfig(validConfig(t), dc), key.String())
		})
	}
}
