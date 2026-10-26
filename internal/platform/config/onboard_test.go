package config_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func onboardEnv() map[string]string {
	return map[string]string{
		"NAVA_DATABASE_URL":            "postgres://nava_app:p@db/nava",
		"NAVA_ONBOARD_DATABASE_URL":    "postgres://nava_onboard:onboard-pw@db/nava",
		"NAVA_OIDC_ISSUERS":            "http://kc/realms/nava",
		"NAVA_ONBOARD_ISSUER":          "http://kc/realms/nava",
		"NAVA_OPERATORS":               "http://kc/realms/nava|op-1",
		"NAVA_KEYCLOAK_URL":            "http://kc",
		"NAVA_DIRECTORY_CLIENT_ID":     "dir",
		"NAVA_DIRECTORY_CLIENT_SECRET": "dir-secret",
		"NAVA_ONBOARD_CLIENT_ID":       "onb",
		"NAVA_ONBOARD_CLIENT_SECRET":   "onboard-secret-value",
	}
}

func lookup(m map[string]string) config.LookupFunc {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestLoadOnboard(t *testing.T) {
	cfg, err := config.LoadOnboard(lookup(onboardEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.KeycloakRealm != "nava" || len(cfg.Operators) != 1 || cfg.Operators[0].Subject != "op-1" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoadOnboardRequiresEverything(t *testing.T) {
	for k := range onboardEnv() {
		t.Run(k, func(t *testing.T) {
			env := onboardEnv()
			delete(env, k)
			_, err := config.LoadOnboard(lookup(env))
			if err == nil || !strings.Contains(err.Error(), k) || strings.Contains(err.Error(), "onboard-secret-value") {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestLoadOnboardRejectsBadOperatorsAndIssuer(t *testing.T) {
	env := onboardEnv()
	env["NAVA_OPERATORS"] = "no-pipe"
	if _, err := config.LoadOnboard(lookup(env)); err == nil {
		t.Error("want error for malformed operators")
	}
	env = onboardEnv()
	env["NAVA_ONBOARD_ISSUER"] = "http://other/realms/x"
	if _, err := config.LoadOnboard(lookup(env)); err == nil {
		t.Error("want error for an issuer outside NAVA_OIDC_ISSUERS")
	}
}

// The tenant-serving process never reads the onboarding secret or the
// nava_onboard database URL: Load ignores both, and nothing of them reaches
// the Config it returns (T21 / S2).
func TestLoadIgnoresOnboardingSecrets(t *testing.T) {
	env := onboardEnv()
	cfg, err := config.Load(lookup(env))
	if err != nil {
		t.Fatal(err)
	}
	dump := fmt.Sprintf("%+v", cfg)
	for _, secret := range []string{"onboard-secret-value", "onboard-pw", "dir-secret"} {
		if strings.Contains(dump, secret) {
			t.Errorf("Config carries %q", secret)
		}
	}
	// A malformed value is not even parsed: nava-core does not read the variable.
	env["NAVA_ONBOARD_DATABASE_URL"] = "not a url"
	if _, err := config.Load(lookup(env)); err != nil {
		t.Errorf("Load read NAVA_ONBOARD_DATABASE_URL: %v", err)
	}
}
