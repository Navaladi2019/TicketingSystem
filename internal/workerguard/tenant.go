// Package workerguard is the worker side of the Temporal tenant path (ADR-001
// A.2 "Guards in the worker", D22): the nava-tenant ContextPropagator, the
// interceptor that enforces every worker guard, the prefixed child-id builder,
// the applied-signal-id set and the activity that closes workflow_ref.
//
// The tenant of a workflow is the <tenant> of its id t:<tenant>:<rest>. The
// relay builds that id from outbox.tenant_id, a row written under RLS (I2), and
// Temporal never changes it. Headers, signal bodies, inputs and search
// attributes are never a source: the guards only compare them with the id.
//
// The Authorizer (temporal-server/authz) checks the same commands again. The
// worker checks first so a bad command fails one workflow, or one activity,
// instead of being denied at the server, where the workflow task retries until
// someone acts.
package workerguard

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// TenantHeader is the Temporal header that carries the tenant, as a
// default-converter string payload (json/plain). temporal-server/authz reads
// the same key.
const TenantHeader = "nava-tenant"

// TenantSearchAttribute is the visibility-only Keyword search attribute. It is
// never read as a tenant; a value set on a child, a continue-as-new or an
// upsert must equal the workflow's tenant, or the Authorizer denies the task.
const TenantSearchAttribute = "NavaTenantId"

// workflowIDPrefix starts every platform workflow id.
const workflowIDPrefix = "t:"

// errNoTenantPrefix is returned for an id that is not t:<canonical uuid>:<rest>.
var errNoTenantPrefix = errors.New("workerguard: workflow id has no t:<tenant>: prefix")

// TenantOf returns the tenant of a t:<tenant>:<rest> workflow id. The tenant
// must be a UUID in canonical form (what uuid.UUID.String writes), so exactly
// one string names each tenant and a string compare with the header is a
// tenant compare.
func TenantOf(workflowID string) (uuid.UUID, error) {
	rest, ok := strings.CutPrefix(workflowID, workflowIDPrefix)
	if !ok {
		return uuid.Nil, errNoTenantPrefix
	}
	s, tail, ok := strings.Cut(rest, ":")
	if !ok || tail == "" {
		return uuid.Nil, errNoTenantPrefix
	}
	tenant, err := uuid.Parse(s)
	if err != nil || tenant == uuid.Nil || tenant.String() != s {
		return uuid.Nil, errNoTenantPrefix
	}
	return tenant, nil
}

// tenantPayload is the header value for tenant: a json/plain string.
func tenantPayload(tenant uuid.UUID) (*commonpb.Payload, error) {
	return converter.GetDefaultDataConverter().ToPayload(tenant.String())
}

// payloadString decodes a default-converter string payload, exactly as the
// Authorizer does: json/plain, data a JSON string. Anything else, including a
// codec-wrapped payload, does not decode.
func payloadString(p *commonpb.Payload) (string, bool) {
	if p == nil || string(p.GetMetadata()[converter.MetadataEncoding]) != converter.MetadataEncodingJSON {
		return "", false
	}
	var s string
	if err := converter.NewJSONPayloadConverter().FromPayload(p, &s); err != nil {
		return "", false
	}
	return s, true
}

// headerOutcome compares the nava-tenant header in fields with tenant. It
// returns outcomeAllow, outcomeMissing (absent or not a json/plain string) or
// outcomeMismatch. The header value itself is never returned: it is not
// logged or recorded anywhere.
func headerOutcome(fields map[string]*commonpb.Payload, tenant uuid.UUID) string {
	got, ok := payloadString(fields[TenantHeader])
	if !ok {
		return outcomeMissing
	}
	if got != tenant.String() {
		return outcomeMismatch
	}
	return outcomeAllow
}
