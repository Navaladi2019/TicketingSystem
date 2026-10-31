// Package interptest holds the interpreter's fixture seams (P3-22a; P3-24
// uses them and does not invent its own, TICK-360): a DefinitionStore and a
// RecordReader read from a directory on every call, and a RecordWriter that
// records every call in that directory. Everything lives on disk, so a test
// process can drive a worker in another process through a mounted volume
// (P3-22b puts them in the stack worker behind a build tag).
//
// Layout under Root:
//
//	<tenant>/definitions/<version id>.json      the definition (def.Parse)
//	<tenant>/definitions/<version id>.ids.json  its step-id map
//	<tenant>/records/<record id>.json           a record
//	writes/calls.jsonl                          one Call per Write
//	writes/applied/<sha256(key)>.json           one file per applied key
//
// It is test support only: depguard refuses it in non-test code.
package interptest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

func definitionPath(root string, tenant, version uuid.UUID, ext string) string {
	return filepath.Join(root, tenant.String(), "definitions", version.String()+ext)
}

func recordPath(root string, tenant, record uuid.UUID) string {
	return filepath.Join(root, tenant.String(), "records", record.String()+".json")
}

func write(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// PutDefinition installs a definition and its step-id map for tenant.
func PutDefinition(root string, tenant, version uuid.UUID, definition, stepIDs []byte) error {
	if err := write(definitionPath(root, tenant, version, ".json"), definition); err != nil {
		return err
	}
	return write(definitionPath(root, tenant, version, ".ids.json"), stepIDs)
}

// PutRecord writes a record of tenant.
func PutRecord(root string, tenant, record uuid.UUID, data map[string]any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return write(recordPath(root, tenant, record), b)
}

// readFile reads path, which must be under root (an os.Root, so no path
// escapes it), and maps a missing file to interp.ErrNotFound.
func readFile(root, path string) ([]byte, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, interp.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	b, err := r.ReadFile(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, interp.ErrNotFound
	}
	return b, err
}

// FixtureStore is an interp.DefinitionStore over Root, read on each Load.
type FixtureStore struct{ Root string }

// Load implements interp.DefinitionStore.
func (s FixtureStore) Load(_ context.Context, tenant, version uuid.UUID) (*interp.Version, error) {
	b, err := readFile(s.Root, definitionPath(s.Root, tenant, version, ".json"))
	if err != nil {
		return nil, err
	}
	d, err := def.Parse(b)
	if err != nil {
		return nil, err
	}
	ids, err := readFile(s.Root, definitionPath(s.Root, tenant, version, ".ids.json"))
	if err != nil {
		return nil, err
	}
	v := &interp.Version{Definition: d}
	if err := json.Unmarshal(ids, &v.StepIDs); err != nil {
		return nil, err
	}
	return v, nil
}

// FileRecords is an interp.RecordReader over Root, read on each call.
type FileRecords struct{ Root string }

// Read implements interp.RecordReader.
func (r FileRecords) Read(_ context.Context, tenant, record uuid.UUID) (map[string]any, error) {
	b, err := readFile(r.Root, recordPath(r.Root, tenant, record))
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Call is one RecordWriter.Write call, as FileWriter records it.
type Call struct {
	Key     string         `json:"key"`
	Attempt int32          `json:"attempt"`
	Tenant  uuid.UUID      `json:"tenant"`
	Record  uuid.UUID      `json:"record"`
	Step    uuid.UUID      `json:"step"`
	Set     map[string]any `json:"set"`
	Applied bool           `json:"applied"`
}

// FileWriter is an interp.RecordWriter that records every call in
// Root/writes/calls.jsonl, with the idempotency key and the activity attempt,
// and applies an effect once per key: the first call creates
// Root/writes/applied/<sha256(key)>.json exclusively, which is atomic across
// processes. Fail, when set, is returned instead of applying (after the call
// is recorded).
type FileWriter struct {
	Root string
	Fail error

	mu sync.Mutex
}

func (w *FileWriter) writesDir() string { return filepath.Join(w.Root, "writes") }

func appliedName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:]) + ".json"
}

// Write implements interp.RecordWriter.
func (w *FileWriter) Write(ctx context.Context, tenant, record, step uuid.UUID, key string, set map[string]any) error {
	c := Call{Key: key, Tenant: tenant, Record: record, Step: step, Set: set}
	if activity.IsActivity(ctx) {
		c.Attempt = activity.GetInfo(ctx).Attempt
	}
	if w.Fail == nil {
		applied, err := w.apply(key)
		if err != nil {
			return err
		}
		c.Applied = applied
	}
	if err := w.appendCall(c); err != nil {
		return err
	}
	return w.Fail
}

// apply creates the key's applied file; false if it already exists.
func (w *FileWriter) apply(key string) (bool, error) {
	dir := filepath.Join(w.writesDir(), "applied")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return false, err
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return false, err
	}
	defer func() { _ = r.Close() }()
	f, err := r.OpenFile(appliedName(key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, werr := f.WriteString(key)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return true, werr
}

func (w *FileWriter) appendCall(c Call) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.MkdirAll(w.writesDir(), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(w.writesDir(), "calls.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(b, '\n'))
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}

// Calls returns every recorded call, in order.
func (w *FileWriter) Calls() ([]Call, error) {
	b, err := os.ReadFile(filepath.Join(w.writesDir(), "calls.jsonl"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Call
	for line := range strings.Lines(string(b)) {
		var c Call
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("interptest: calls.jsonl: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// Applications is how many times key's effect was applied: 0 or 1.
func (w *FileWriter) Applications(key string) (int, error) {
	_, err := os.Stat(filepath.Join(w.writesDir(), "applied", appliedName(key)))
	switch {
	case err == nil:
		return 1, nil
	case errors.Is(err, fs.ErrNotExist):
		return 0, nil
	}
	return 0, err
}

var (
	_ interp.DefinitionStore = FixtureStore{}
	_ interp.RecordReader    = FileRecords{}
	_ interp.RecordWriter    = (*FileWriter)(nil)
)
