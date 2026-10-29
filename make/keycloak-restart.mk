.PHONY: stack-test-keycloak_restart

# TICK-328 (R4): restarts PostgreSQL and requires Keycloak to issue tokens again
# (stack-test.sh keycloak_restart). Listed in STACK_TESTS_SERIAL in stack.mk,
# after dependencies, before the disruptive cases.
