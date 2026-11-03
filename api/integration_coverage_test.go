package api_test

// TICK-510 gate, red step: the fixtures in integration_coverage_fixture_test.go
// come first. These stubs only make the package compile; the implementation
// commit replaces them with the gate.

type coverMarker struct {
	op, status, where string
}

func scanCoverMarkers(string, []byte) ([]coverMarker, []string) { return nil, nil }

func checkCoverage(map[string]map[string]string, []coverMarker, map[string]map[string]string) []string {
	return nil
}
