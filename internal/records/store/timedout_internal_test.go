package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TICK-649: only a DataFilter list's 57014 is the data filter's timeout. A
// cancel on a list without one (a role-level timeout, pg_cancel_backend) and a
// cancel because the request itself ended keep the raw error.
func TestTimedOutMapsOnlyADataFilterCancel(t *testing.T) {
	cancel := &pgconn.PgError{Code: "57014"}
	other := &pgconn.PgError{Code: "40001"}
	live := context.Background()
	ended, stop := context.WithCancel(context.Background())
	stop()

	for _, tc := range []struct {
		name       string
		ctx        context.Context
		dataFilter bool
		err        error
		want       bool
	}{
		{"57014 with a data filter", live, true, cancel, true},
		{"57014 without a data filter", live, false, cancel, false},
		{"57014 after the request ended", ended, true, cancel, false},
		{"another SQLSTATE with a data filter", live, true, other, false},
		{"a non-database error with a data filter", live, true, errors.New("boom"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := timedOut(tc.ctx, ListQuery{DataFilter: tc.dataFilter}, tc.err)
			if is := errors.Is(got, ErrStatementTimeout); is != tc.want {
				t.Errorf("errors.Is(ErrStatementTimeout) = %v, want %v (err %v)", is, tc.want, got)
			}
			if !errors.Is(got, tc.err) {
				t.Errorf("the original error is lost: %v", got)
			}
		})
	}
}
