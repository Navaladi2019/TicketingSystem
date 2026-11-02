package def

import "github.com/google/uuid"

// SignalKeyPrefix prefixes a wait_signal name in the step-id map, so signal
// ids and step ids share one map without colliding (a step key cannot
// contain ':').
const SignalKeyPrefix = "signal:"

// SignalKey is the step-id map key for a wait_signal name.
func SignalKey(name string) string { return SignalKeyPrefix + name }

// AssignStepIDs returns key -> id for every step key, plus "signal:<name>" ->
// id for every distinct wait_signal name. It keeps prev's id for every key
// that exists in prev and draws uuid.New() (random v4 from crypto/rand) for
// every new key. Ids are never derived from a key (TICK-363), so two tenants'
// identical keys get unrelated ids. prev entries for keys no longer in def
// are dropped. Pure apart from the random source.
//
// P3-41 calls it server-side at publish and stores the map in the immutable
// version row; a client never supplies a step id.
func AssignStepIDs(def *Definition, prev map[string]uuid.UUID) map[string]uuid.UUID {
	names := def.SignalNames()
	out := make(map[string]uuid.UUID, len(def.Steps)+len(names))
	assign := func(k string) {
		if id, ok := prev[k]; ok && id != uuid.Nil {
			out[k] = id
			return
		}
		out[k] = uuid.New()
	}
	for i := range def.Steps {
		assign(def.Steps[i].Key)
	}
	for _, n := range names {
		assign(SignalKey(n))
	}
	return out
}
