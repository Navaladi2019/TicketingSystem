package authz

import "strconv"

// PlantedErrcheck is a deliberate errcheck violation (TICK-584 acceptance):
// the temporal-server job must go red on it. Reverted before merge.
func PlantedErrcheck() {
	strconv.Atoi("1")
}
