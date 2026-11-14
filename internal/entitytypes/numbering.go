package entitytypes

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// The number-format link of a definition publish (ADR-005 §3.2, §3.3, §3.7
// C8, C10; TICK-673, P6-08). The link is a pinned (format id, format
// version) on the new definition version, with the compiled form
// snapshotted beside it, so a record create never reads the format tables.

// ErrFormatNotFound means the format version the publish names does not
// exist in the tenant. Another tenant's format is the same error: the read
// runs under the tenant's row-level security, and the composite foreign key
// leading with tenant_id is the backstop (C8).
var ErrFormatNotFound = errors.New("entitytypes: number format not found")

// formatFKConstraint is the composite link key of entity_definition_versions.
const formatFKConstraint = "entity_definition_versions_number_format_fkey"

// OverlapError is returned by Publish when the format's compiled template
// could render a number an earlier epoch of the type already issued
// (numbering.Disjoint, ADR-005 §3.3). Template is the clashing epoch's
// template. Nothing is written.
type OverlapError struct{ Template string }

func (e *OverlapError) Error() string {
	return "entitytypes: number format overlaps epoch " + strconv.Quote(e.Template)
}

// ConflictError is returned by Publish when ExpectedVersion does not hold
// the type's current version. Nothing is written.
type ConflictError struct{ Current int }

func (e *ConflictError) Error() string {
	return fmt.Sprintf("entitytypes: entity type is at version %d", e.Current)
}

// FormatRef is a pinned number-format version.
type FormatRef struct {
	ID      uuid.UUID
	Version int
}

// NumberFormatChange says what a publish does to the type's number-format
// link. The zero value keeps the previous version's link: an unrelated
// publish never changes numbering. Set with a nil Ref unpins (the default
// numbering); Set with a Ref pins that format version.
type NumberFormatChange struct {
	Set bool
	Ref *FormatRef
}

// link is what the new version row stores: all nil is the default.
type link struct {
	id       *uuid.UUID
	version  *int
	compiled *string
}

// resolveLink works out the link of the version about to be written, taking
// or reusing an epoch for a pinned format. It must run after the type's
// row is locked FOR UPDATE (C10).
func resolveLink(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, prefix string, previous int, ch NumberFormatChange) (link, error) {
	switch {
	case !ch.Set:
		var l link
		if err := tx.QueryRow(ctx, `
SELECT number_format_id, number_format_version, number_compiled::text
FROM entity_definition_versions WHERE definition_id = $1 AND version = $2`, defID, previous).
			Scan(&l.id, &l.version, &l.compiled); err != nil {
			return link{}, fmt.Errorf("entitytypes: read pinned format: %w", err)
		}
		return l, nil
	case ch.Ref == nil:
		return link{}, nil
	}
	var compiledText string
	err := tx.QueryRow(ctx, `
SELECT compiled::text FROM number_format_versions WHERE format_id = $1 AND version = $2`,
		ch.Ref.ID, ch.Ref.Version).Scan(&compiledText)
	if errors.Is(err, pgx.ErrNoRows) {
		return link{}, ErrFormatNotFound
	}
	if err != nil {
		return link{}, fmt.Errorf("entitytypes: read number format: %w", err)
	}
	var c numbering.Compiled
	if err := json.Unmarshal([]byte(compiledText), &c); err != nil {
		return link{}, fmt.Errorf("entitytypes: stored compiled number format: %w", err)
	}
	if c, err = takeEpoch(ctx, tx, tenantID, defID, prefix, c); err != nil {
		return link{}, err
	}
	snap, err := json.Marshal(c)
	if err != nil {
		return link{}, fmt.Errorf("entitytypes: encode compiled number format: %w", err)
	}
	text := string(snap)
	id, version := ch.Ref.ID, ch.Ref.Version
	return link{id: &id, version: &version, compiled: &text}, nil
}

// takeEpoch returns c with its epoch: 0 for the default, the type's existing
// epoch for a template it has used before (re-adopting a template continues
// its counter), else the next one after the disjointness check against every
// epoch of the type, epoch 0 included (§3.3).
func takeEpoch(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, prefix string, c numbering.Compiled) (numbering.Compiled, error) {
	if c.Epoch == 0 {
		return c, nil
	}
	head, err := c.LiteralHead(prefix) // also refuses a compiled form that does not match its template
	if err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: stored compiled number format: %w", err)
	}
	sum, err := hex.DecodeString(c.CompiledSHA256)
	if err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: compiled hash is not hex: %w", err)
	}
	var epoch int
	switch err := tx.QueryRow(ctx, `
SELECT epoch FROM numbering_epochs WHERE entity_definition_id = $1 AND compiled_sha256 = $2`, defID, sum).Scan(&epoch); {
	case err == nil:
		c.Epoch = epoch
		return c, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return numbering.Compiled{}, fmt.Errorf("entitytypes: read epoch: %w", err)
	}
	zero, err := numbering.Default().LiteralHead(prefix)
	if err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: default head: %w", err)
	}
	if !numbering.DisjointHeads(zero, head) {
		return numbering.Compiled{}, &OverlapError{Template: templateText(numbering.Default())}
	}
	rows, err := tx.Query(ctx, `
SELECT compiled_sha256, literal_head FROM numbering_epochs WHERE entity_definition_id = $1 ORDER BY epoch`, defID)
	if err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: read epochs: %w", err)
	}
	var clash []byte
	for rows.Next() {
		var other []byte
		var otherHead string
		if err := rows.Scan(&other, &otherHead); err != nil {
			rows.Close()
			return numbering.Compiled{}, fmt.Errorf("entitytypes: scan epoch: %w", err)
		}
		if !numbering.DisjointHeads(otherHead, head) {
			clash = other
			break
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: read epochs: %w", err)
	}
	if clash != nil {
		return numbering.Compiled{}, &OverlapError{Template: epochTemplate(ctx, tx, defID, clash)}
	}
	if err := tx.QueryRow(ctx, `
SELECT coalesce(max(epoch), 0) + 1 FROM numbering_epochs WHERE entity_definition_id = $1`, defID).Scan(&epoch); err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: next epoch: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head)
VALUES ($1, $2, $3, $4, $5)`, tenantID, defID, epoch, sum, head); err != nil {
		return numbering.Compiled{}, fmt.Errorf("entitytypes: insert epoch: %w", err)
	}
	c.Epoch = epoch
	return c, nil
}

// templateText is the template text of c: its head, {seq:N} and tail.
func templateText(c numbering.Compiled) string {
	return c.Head + "{seq:" + strconv.Itoa(c.Width) + "}" + c.Tail
}

// epochTemplate names the epoch with the given hash by the template of a
// version that carries its snapshot. The registry stores no template, so a
// version that no longer reads falls back to the hash.
func epochTemplate(ctx context.Context, tx pgx.Tx, defID uuid.UUID, sum []byte) string {
	want := hex.EncodeToString(sum)
	rows, err := tx.Query(ctx, `
SELECT number_compiled::text FROM entity_definition_versions
WHERE definition_id = $1 AND number_compiled IS NOT NULL ORDER BY version DESC`, defID)
	if err != nil {
		return want
	}
	defer rows.Close()
	for rows.Next() {
		var text string
		var c numbering.Compiled
		if rows.Scan(&text) == nil && json.Unmarshal([]byte(text), &c) == nil && c.CompiledSHA256 == want {
			return templateText(c)
		}
	}
	return want
}

// mapFormatFK turns a violation of the format link key into ErrFormatNotFound.
func mapFormatFK(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23503" && pe.ConstraintName == formatFKConstraint {
		return ErrFormatNotFound
	}
	return err
}

// LinkedBelow returns the keys of the entity types whose current version pins
// a version of format below version, by key: what "re-publish linked types"
// acts on. It reads in the caller's InTenantTx, so only the tenant's types
// appear.
func (Store) LinkedBelow(ctx context.Context, tx pgx.Tx, format uuid.UUID, version int) ([]string, error) {
	rows, err := tx.Query(ctx, `
SELECT d.key
FROM entity_definitions d
JOIN entity_definition_versions v
  ON v.tenant_id = d.tenant_id AND v.definition_id = d.id AND v.version = d.current_version
WHERE v.number_format_id = $1 AND v.number_format_version < $2
ORDER BY d.key`, format, version)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: linked types: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("entitytypes: scan linked type: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("entitytypes: linked types: %w", err)
	}
	return keys, nil
}
