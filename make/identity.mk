.PHONY: stack-test-identity identity-attacks identity-attacks-fixture stack-test-onboard stack-idp-realms

# Creates the two test IdP realms (idp-tenant-a, idp-tenant-b) on the running
# Keycloak, fresh, with the code the identity tests start with (TICK-166). A
# plain `docker compose up` imports only `nava`; a manual S2 step needs these.
stack-idp-realms:
	go test -tags identitystack -count=1 -run ^TestIdPRealmsExist$$ ./internal/identitystack

# The realm `nava` of the running Keycloak (T14, ADR-001 A.1): tenant and idp
# claims, token lifespan, operator role custody, realm export round trip.
# TestCI5 needs a platform database: make identity-attacks runs it. The
# CI-5.06 negative fixture needs Keycloak 26.1.2: make identity-attacks-fixture.
# It skips the onboarding tests below (ONBOARD_TESTS): stack-test-onboard runs
# them, the onboard case (TICK-628, TODO-014). Both declare the one pattern to
# suiteran too (-skip here, -run there), so each checks its own half of the
# package and every test runs in exactly one of the two.
# suiteran fails the target on its own when a registered suite ran zero tests
# (TICK-283); TestCI5 is behind the ci5 tag, so it is not expected here. The
# stream is kept for make identity-attacks, which checks it with the CI-5 one.
stack-test-identity:
	rm -f $(IDENTITY_STREAM)
	go run ./tools/ci/suiteran -tags identitystack -skip '$(ONBOARD_TESTS)' -tee $(IDENTITY_STREAM) -- go test -tags identitystack -count=1 -skip '$(ONBOARD_TESTS)' -json ./internal/identitystack

# The CI-5 adversarial identity suite (T19, ADR-001 Annex B CI-5) against the
# running Keycloak, with a throwaway PostgreSQL for the platform database the
# resolver and one real group sync use. Writes the CI-5 report to CI5_REPORT
# and fails when a CI-5 case did not pass. Needs the stack's Keycloak.
# ONBOARD_TESTS: the stack stream skipped them (stack-test-onboard runs them),
# so suiteran does not expect them there either.
CI5_REPORT ?= ci5-report.md
IDENTITY_STREAM ?= identity-stream.json
identity-attacks:
	CI5_REPORT=$(CI5_REPORT) IDENTITY_STREAM=$(IDENTITY_STREAM) ONBOARD_TESTS='$(ONBOARD_TESTS)' sh scripts/ci/identity-attacks.sh

# The CI-5.06 negative fixture: the same kc.org tamper must SUCCEED on a
# throwaway Keycloak 26.1.2 (the known-broken pin), so CI-5.06 passing on the
# pinned Keycloak means something. Needs Docker, not the stack. Opt-in.
identity-attacks-fixture:
	sh scripts/ci/identity-attacks.sh fixture

# Onboarding end to end (T21b, TICK-234): nava-onboard onboards a tenant, the
# invited admin signs in through the tenant's IdP, one nava-groupsync cycle
# runs, and nava-core's GET /api/v1/me shows the new tenant and no other; the
# nava-onboard/nava-core separation; kill before the binding, then retry,
# against the real nava_onboard PGStore; and the onboarded tenant sealed by
# the always-on nava-worker (TICK-453). Needs the stack up (make stack-up).
# Under suiteran (TICK-628): onboard_invite_test.go holds cross-tenant cases,
# and this is the only place they run.
ONBOARD_TESTS := ^(TestOnboard(edAdminSignsInAndMeShowsTenant|edTenantIsSealed|DeploymentSeparation|KillBeforeBindThenRetryLiveStack|DomainClaimedIsConflict)|TestInvitedToTenantANotLinkedThroughTenantBIdP|TestSquattedInviteeIsNotPlacedInTheOtherTenant|TestUninvitedLocalUserNotLinked|TestInviteIsSingleUse|TestExpiredInviteNotLinked|TestInvitedUserInTwoTenantsNotLinked|TestInvitedUserHasNoLocalLogin|TestDisabledInviteeNotLinked|TestUnverifiedUpstreamEmailNotPlaced|TestIdPEmailVerifiedAttributeNotUserEditable|TestUserCannotRenameEmailToPendingInvite|TestAccountAPIClosedToPlainBrokeredUser|TestRealmFileEmailVerifiedMapper)$$
stack-test-onboard:
	go run ./tools/ci/suiteran -tags identitystack -run '$(ONBOARD_TESTS)' -- go test -tags identitystack -count=1 -run '$(ONBOARD_TESTS)' -json ./internal/identitystack

