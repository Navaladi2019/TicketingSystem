package tenantsettings

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errBoom = errors.New("injected database fault")

// stubTx answers the two calls the store makes before the audit write. The
// write path through the audit entry runs against PostgreSQL in
// internal/server (tenantsettings_integration_test.go).
type stubTx struct {
	pgx.Tx
	zone    string
	rowErr  error
	execErr error
	rows    int64
	execs   int
}

func (s *stubTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return zoneRow{s.zone, s.rowErr}
}

func (s *stubTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	s.execs++
	if s.execErr != nil {
		return pgconn.CommandTag{}, s.execErr
	}
	return pgconn.NewCommandTag("UPDATE " + string(rune('0'+s.rows))), nil
}

type zoneRow struct {
	zone string
	err  error
}

func (r zoneRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*string)) = r.zone
	return nil
}

func TestCheckTimeZone(t *testing.T) {
	t.Parallel()
	for _, ok := range []string{"UTC", "Europe/Berlin", "America/Argentina/Buenos_Aires", "Etc/GMT+5"} {
		if err := CheckTimeZone(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Local", " UTC", "UTC ", "Mars/Base", "../etc/passwd", strings.Repeat("A", 65)} {
		if err := CheckTimeZone(bad); !errors.Is(err, ErrInvalidTimeZone) {
			t.Errorf("%q = %v, want ErrInvalidTimeZone", bad, err)
		}
	}
}

func TestTimeZoneRead(t *testing.T) {
	t.Parallel()
	if got, err := (Store{}).TimeZone(t.Context(), &stubTx{zone: "Asia/Tokyo"}, uuid.New()); err != nil || got != "Asia/Tokyo" {
		t.Errorf("= %q, %v", got, err)
	}
	if _, err := (Store{}).TimeZone(t.Context(), &stubTx{rowErr: pgx.ErrNoRows}, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("no row = %v", err)
	}
	if _, err := (Store{}).TimeZone(t.Context(), &stubTx{rowErr: errBoom}, uuid.New()); !errors.Is(err, errBoom) || errors.Is(err, ErrNotFound) {
		t.Errorf("fault = %v", err)
	}
}

func TestSetTimeZoneRefusals(t *testing.T) {
	t.Parallel()
	by := uuid.New()
	t.Run("no actor", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{zone: "UTC"}
		if _, err := (Store{}).SetTimeZone(t.Context(), tx, uuid.New(), "Asia/Tokyo", uuid.Nil); !errors.Is(err, ErrNoActor) || tx.execs != 0 {
			t.Errorf("err = %v, %d statements", err, tx.execs)
		}
	})
	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{zone: "UTC"}
		if _, err := (Store{}).SetTimeZone(t.Context(), tx, uuid.New(), "Mars/Base", by); !errors.Is(err, ErrInvalidTimeZone) || tx.execs != 0 {
			t.Errorf("err = %v, %d statements", err, tx.execs)
		}
	})
	t.Run("read fails", func(t *testing.T) {
		t.Parallel()
		if _, err := (Store{}).SetTimeZone(t.Context(), &stubTx{rowErr: errBoom}, uuid.New(), "Asia/Tokyo", by); !errors.Is(err, errBoom) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("same value writes nothing", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{zone: "Asia/Tokyo"}
		got, err := (Store{}).SetTimeZone(t.Context(), tx, uuid.New(), "Asia/Tokyo", by)
		if err != nil || got != "Asia/Tokyo" || tx.execs != 0 {
			t.Errorf("= %q, %v, %d statements", got, err, tx.execs)
		}
	})
	t.Run("update fails", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{zone: "UTC", execErr: errBoom}
		if _, err := (Store{}).SetTimeZone(t.Context(), tx, uuid.New(), "Asia/Tokyo", by); !errors.Is(err, errBoom) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("update matches no row", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{zone: "UTC", rows: 0}
		if _, err := (Store{}).SetTimeZone(t.Context(), tx, uuid.New(), "Asia/Tokyo", by); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v", err)
		}
	})
}
