.PHONY: stack-test-keycloak_restart

# TICK-328 (R4): restarts PostgreSQL and requires Keycloak to issue tokens again.
# Listed in STACK_TESTS_SERIAL in stack.mk, after dependencies, before the
# disruptive cases. The recipe is explicit: make does not apply the
# stack-test-% pattern rule to a .PHONY target, so without it the target is a
# silent no-op.
stack-test-keycloak_restart:
	sh deploy/compose/test/stack-test.sh keycloak_restart
