.PHONY: stack-test-keycloak_restart

# TICK-328 (R4): restarts PostgreSQL and requires Keycloak to issue tokens again
# (stack-test.sh keycloak_restart). Runs among the serial cases, before the
# disruptive ones in stack.mk.
STACK_TESTS_SERIAL += keycloak_restart
