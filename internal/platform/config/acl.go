package config

import (
	"fmt"
	"strconv"
)

// The platform-wide byte limit on an access policy's rules and masks (ADR-010
// amendment A1, A1-2a): measured on the jsonb text form, so that the route and
// the database count the same bytes.
const (
	// DefaultACLMaxDocumentBytes is 32 KiB.
	DefaultACLMaxDocumentBytes = 32768
	// MinACLMaxDocumentBytes is the lowest setting, 16 KiB.
	MinACLMaxDocumentBytes = 16384
	// MaxACLMaxDocumentBytes is the highest setting, 128 KiB: the database's
	// own ceiling (acl_versions_bytes_rules_check and _masks_check).
	MaxACLMaxDocumentBytes = 131072
)

// ACL configures the access-policy publish path.
type ACL struct {
	// MaxDocumentBytes is NAVA_ACL_MAX_DOCUMENT_BYTES, an integer from 16,384
	// to 131,072; unset is 32,768. nava-core does not start on any other
	// value. It is not a public setting: no route accepts or returns it,
	// except as the limit in a document_too_large problem's detail.
	MaxDocumentBytes int
}

// acl reads the access-policy variables. Errors name the variable only.
func (r *reader) acl() ACL {
	const key = "NAVA_ACL_MAX_DOCUMENT_BYTES"
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return ACL{MaxDocumentBytes: DefaultACLMaxDocumentBytes}
	}
	n, err := strconv.Atoi(v)
	if err != nil || strconv.Itoa(n) != v || n < MinACLMaxDocumentBytes || n > MaxACLMaxDocumentBytes {
		r.errs = append(r.errs, fmt.Errorf("%s: want an integer from %d to %d", key, MinACLMaxDocumentBytes, MaxACLMaxDocumentBytes))
		return ACL{MaxDocumentBytes: DefaultACLMaxDocumentBytes}
	}
	return ACL{MaxDocumentBytes: n}
}
