//go:build integration

package seal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
)

func loadVectors(t *testing.T) vectors.File {
	t.Helper()
	b, err := os.ReadFile("../chain/testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var f vectors.File
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// seedVector inserts one chain vector's row exactly as it stands before
// sealing: its envelope, its stored diff and salt, and (for an erased entry)
// its erasure columns. The vector's occurred_at, id and backfilled are
// columns R10 lets only the migrator write, so the superuser seeds them with
// session_replication_role = replica, which skips R10 and the foreign-key
// triggers on this connection only. The sealer then runs as nava_worker with
// every trigger on.
func seedVector(t *testing.T, e itEnv, v vectors.ChainCase) {
	t.Helper()
	env := v.Envelope
	hash := func(h *[32]byte) []byte {
		if h == nil {
			return nil
		}
		return h[:]
	}
	var salt []byte
	if v.DiffSalt != nil {
		s, err := chain.ParseSalt16(*v.DiffSalt)
		if err != nil {
			t.Fatal(err)
		}
		salt = s[:]
	}
	var erasedAt *time.Time
	if v.ErasedAt != nil {
		at, err := time.Parse(chain.TimeLayout, *v.ErasedAt)
		if err != nil {
			t.Fatal(err)
		}
		erasedAt = &at
	}
	var erasureRef *uuid.UUID
	if v.ErasureRef != nil {
		ref := uuid.MustParse(*v.ErasureRef)
		erasureRef = &ref
	}
	ctx := context.Background()
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type,
		subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source,
		source_ref, change_set_id, causation_id, causation_depth, request_id, backfilled, content_sha256,
		diff_sha256, diff_truncated, diff, diff_salt, erased_at, erasure_ref)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		$21, $22, $23::json, $24, $25, $26)`,
		env.TenantID, env.ID, env.OccurredAt, env.Domain, env.Action, env.SubjectType, env.SubjectID,
		env.EntityDefinitionID, env.VersionBefore, env.VersionAfter, env.ActorPrincipalID, env.ActorSystem,
		env.Source, env.SourceRef, env.ChangeSetID, env.CausationID, env.CausationDepth, env.RequestID,
		env.Backfilled, hash(env.ContentSHA256), hash(env.DiffSHA256), env.DiffTruncated, v.Diff, salt,
		erasedAt, erasureRef); err != nil {
		t.Fatalf("seed %s: %v", v.Name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

// AC: sealed rows hash to P5-02's golden vectors. The vector chain is seeded
// in the order its entries would commit: seq 4 is a back-filled entry
// (occurred_at 1999, R25) that arrives after seq 1 to 3 are sealed, so the
// sealer runs once per arrival. Every sealed row's seq, prev_hash and
// entry_hash must equal the vector's, and a nava.audit.export/v1 file built
// from the sealed range by SQL must verify (nava audit verify: pass).
func TestSealGoldenVectors(t *testing.T) {
	e := migrated(t)
	f := loadVectors(t)
	tenant := f.Chain[0].Envelope.TenantID
	seedHead(t, e, tenant)
	ctx := asTenant(tenant)

	for _, arrival := range [][]int64{{1, 2, 3}, {4}, {5, 6, 7}} {
		for _, seq := range arrival {
			seedVector(t, e, f.Chain[seq-1])
		}
		n, err := seal.Seal(ctx, e.worker)
		if err != nil || n != len(arrival) {
			t.Fatalf("Seal after seq %v arrived = %d, %v", arrival, n, err)
		}
	}
	if len(f.Chain) != 7 {
		t.Fatalf("the vectors have %d chain cases; this test seeds 7", len(f.Chain))
	}

	for _, v := range f.Chain {
		var (
			seq        int64
			prev, hash []byte
		)
		if err := e.super.QueryRow(context.Background(), `SELECT seq, prev_hash, entry_hash FROM audit_log
			WHERE tenant_id = $1 AND id = $2`, tenant, v.Envelope.ID).Scan(&seq, &prev, &hash); err != nil {
			t.Fatal(err)
		}
		if seq != v.Seq || chain.Hex(prev) != v.PrevHash || chain.Hex(hash) != v.EntryHash {
			t.Errorf("%s: sealed as seq %d prev %x entry %x; want the vector's seq %d prev %s entry %s",
				v.Name, seq, prev, hash, v.Seq, v.PrevHash, v.EntryHash)
		}
	}
	last := f.Chain[len(f.Chain)-1]
	if h := readHead(t, e, tenant); h.seq != last.Seq || chain.Hex(h.entryHash) != last.EntryHash {
		t.Errorf("head at seq %d %x, want the vector's last, seq %d %s", h.seq, h.entryHash, last.Seq, last.EntryHash)
	}

	rep, err := verify.Verify(bytes.NewReader(exportBySQL(t, e, tenant)))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Outcome != verify.OutcomePass {
		t.Errorf("verify: %s %+v, want pass", rep.Outcome, rep.Failure)
	}
	if rep.Count != len(f.Chain) || rep.LastSeq == nil || *rep.LastSeq != last.Seq {
		t.Errorf("verify read %d entries to seq %v, want %d to %d", rep.Count, rep.LastSeq, len(f.Chain), last.Seq)
	}
}

// exportBySQL writes the tenant's whole sealed range as a stubs=true
// nava.audit.export/v1 JSONL file (audit-export-v1.md §3), from genesis,
// every line read from the database. The line shapes are the spec's; the
// export endpoint itself is a later task.
func exportBySQL(t *testing.T, e itEnv, tenant uuid.UUID) []byte {
	t.Helper()
	type obj = map[string]any
	null := (*string)(nil)
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	write := func(v any) {
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
	}
	write(obj{
		"kind": "header", "format": "nava.audit.export/v1", "tenant_id": tenant.String(),
		"export_id": uuid.Must(uuid.NewV7()).String(), "exported_at": time.Now().UTC().Truncate(time.Microsecond).Format(chain.TimeLayout),
		"stubs": true,
		"filter": obj{"from": null, "to": null, "domain": null, "subject_type": null, "subject_id": null,
			"change_set_id": null, "after_seq": null},
	})
	write(obj{"kind": "anchor", "genesis": true, "seq": 0, "prev_hash": null,
		"entry_hash": chain.Hex(chain.Genesis[:]), "envelope_sha256": null})

	rows, err := e.super.Query(context.Background(), `SELECT seq, prev_hash, entry_hash, sealed_at, diff::text,
		diff_salt, erased_at, erasure_ref, `+envelopeColumns+`
		FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NOT NULL ORDER BY seq`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var (
		count    int
		lastSeq  int64
		headHash string
	)
	for rows.Next() {
		var (
			seq              int64
			prev, hash, salt []byte
			sealedAt         time.Time
			diff             *string
			erasedAt         *time.Time
			erasureRef       *uuid.UUID
		)
		env := scanEnvelope(t, rows, tenant, &seq, &prev, &hash, &sealedAt, &diff, &salt, &erasedAt, &erasureRef)
		sum, err := chain.EnvelopeSHA256(env)
		if err != nil {
			t.Fatal(err)
		}
		line := obj{
			"kind": "entry", "sealed": true, "in_verified_range": true, "seq": seq,
			"prev_hash": chain.Hex(prev), "entry_hash": chain.Hex(hash), "envelope_sha256": chain.Hex(sum[:]),
			"sealed_at": sealedAt.UTC().Format(chain.TimeLayout), "envelope": env,
			"diff": diff, "diff_salt": null, "diff_redacted": false, "erased_at": null, "erasure_ref": null,
		}
		if salt != nil {
			line["diff_salt"] = chain.Hex(salt)
		}
		if erasedAt != nil {
			line["erased_at"] = erasedAt.UTC().Format(chain.TimeLayout)
		}
		if erasureRef != nil {
			line["erasure_ref"] = erasureRef.String()
		}
		write(line)
		count++
		lastSeq, headHash = seq, chain.Hex(hash)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	var anchor int64
	write(obj{"kind": "trailer", "anchor_seq": anchor, "last_seq": lastSeq, "head_hash": headHash,
		"count": count, "stub_count": 0, "unsealed_count": 0, "verifiable": true})
	return out.Bytes()
}
