package workerguard

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	commonpb "go.temporal.io/api/common/v1"
)

var errMeter = errors.New("meter refused the instrument")

// failingMeterProvider hands out a meter whose Int64Counter fails.
type failingMeterProvider struct{ metricnoop.MeterProvider }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return failingMeter{} }

type failingMeter struct{ metricnoop.Meter }

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errMeter
}

// A meter that cannot register the decisions counter fails the guard's
// setup instead of running it unmetered.
func TestNewRecorderMeterError(t *testing.T) {
	rec, err := newRecorder(slog.New(slog.DiscardHandler), failingMeterProvider{})
	if !errors.Is(err, errMeter) || rec != nil {
		t.Fatalf("newRecorder = %v, %v; want nil, %v", rec, err, errMeter)
	}
	g, err := New(slog.New(slog.DiscardHandler), failingMeterProvider{})
	if !errors.Is(err, errMeter) || g != nil {
		t.Fatalf("New = %v, %v; want nil, %v", g, err, errMeter)
	}
}

// A value the data converter cannot encode is never recorded, as a result or
// as a SideEffect value.
func TestUnencodableValueIsDenied(t *testing.T) {
	ch := make(chan int)
	if resultOK(ch, false) || resultOK(ch, true) {
		t.Error("resultOK recorded a channel")
	}
	if sideEffectValueOK(ch) {
		t.Error("sideEffectValueOK recorded a channel")
	}
}

// A message shorter than the rejection prefix is not a rejection.
func TestShortRejectionMessage(t *testing.T) {
	for _, msg := range []string{"", "x", rejectionPrefix[:len(rejectionPrefix)-1]} {
		if rejectionMessageOK(msg) {
			t.Errorf("%q passed as a rejection", msg)
		}
	}
}

// sameInput refuses a payload list of another length before comparing any
// payload: a dropped or added argument is a different input.
func TestSameInputLength(t *testing.T) {
	ps := []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`"x"`)}}
	in := []checkedPayload{{metadata: map[string][]byte{"encoding": []byte("json/plain")}, data: []byte(`"x"`)}}
	if !sameInput(ps, in) {
		t.Fatal("identical inputs differ")
	}
	if sameInput(ps, nil) {
		t.Error("one payload against none is the same input")
	}
	if sameInput(nil, in) {
		t.Error("no payload against one is the same input")
	}
	if sameInput(append(ps, ps[0]), in) {
		t.Error("an extra payload is the same input")
	}
}

// Malformed JSON objects are refused by directiveFields' own token checks,
// not only by the field rules.
func TestDirectiveEncodingMalformed(t *testing.T) {
	step := uuid.NewString()
	for name, enc := range map[string]string{
		"comma for a key":    `{,"step":"` + step + `"}`,
		"key with no value":  `{"step":}`,
		"unterminated":       `{"step":"` + step + `","kind":"end","wait_seconds":0`,
		"closed by bracket":  `{"step":"` + step + `","kind":"end","wait_seconds":0]`,
		"empty":              ``,
		"value is truncated": `{"step":"` + step,
	} {
		if directiveOK([]byte(enc)) {
			t.Errorf("%s: accepted: %q", name, enc)
		}
	}
}

// directiveResultOK records a Directive only when it encodes as JSON. A nil
// *Directive encodes as binary/null, so it is refused; a valid one is not.
func TestDirectiveResultOKNeedsJSONEncoding(t *testing.T) {
	d := validDirectives()[KindEnd]
	if !directiveResultOK(d) || !directiveResultOK(&d) {
		t.Fatalf("a valid Directive was refused: %+v", d)
	}
	if directiveResultOK((*Directive)(nil)) {
		t.Error("a nil *Directive was recorded")
	}
	if resultOK((*Directive)(nil), true) {
		t.Error("resultOK recorded a nil *Directive on the directive path")
	}
}
