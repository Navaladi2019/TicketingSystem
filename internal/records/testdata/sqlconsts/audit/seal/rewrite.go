package seal

import "context"

// The TICK-581 bypass: SQL allowed in seal.go, run from a sibling file.
func rewrite(ctx context.Context, tx execer) error {
	if err := tx.Exec(ctx, sealSQL); err != nil { // want: update audit_log
		return err
	}
	q := rewriteVar                        // want: update audit_log
	return tx.Exec(ctx, q+" RETURNING id") // look-alike: q is a local, not SQL the gate can read
}

type holder struct{ sealSQL string }

// Look-alike: a field named like the constant is not the constant.
func field(h holder) string {
	return h.sealSQL
}
