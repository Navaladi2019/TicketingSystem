package authz

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/metrics"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	// Register the service descriptors used to find namespace-scoped requests.
	_ "go.temporal.io/api/operatorservice/v1"
	_ "go.temporal.io/api/workflowservice/v1"
	_ "go.temporal.io/server/api/adminservice/v1"
)

// DeniedMetric counts denied calls, tagged identity, api, namespace, reason and
// command (the denied workflow command's type, or "none" for an API deny).
const DeniedMetric = "nava_authz_denied"

// Deny reasons, used as the `reason` log field and metric tag.
const (
	ReasonNoIdentity = "no_identity"
	ReasonAPI        = "api_not_allowed"
	ReasonNamespace  = "namespace_not_allowed"
	ReasonInternal   = "internal_error"
)

// maxLoggedNamespace bounds the client-supplied namespace written to the log.
const maxLoggedNamespace = 128

// maxLoggedWorkflowID bounds a client-supplied workflow id written to the log.
const maxLoggedWorkflowID = 256

var (
	allow = authorization.Result{Decision: authorization.DecisionAllow}
	// No Reason: the caller gets the server's generic "Request unauthorized".
	// The reason goes to the log and the metric only.
	deny = authorization.Result{Decision: authorization.DecisionDeny}
)

// Authorizer is the deny-by-default frontend Authorizer.
type Authorizer struct {
	logger  log.Logger
	metrics metrics.Handler
	allowed map[Identity]map[string]struct{}
	// known is every classified API; anything else is tagged "other" in metrics.
	known map[string]struct{}
	// namespaceScoped is the set of allow-listed APIs whose request has a
	// `namespace` field. Those must target Namespace; the rest must target none.
	namespaceScoped map[string]bool
}

var _ authorization.Authorizer = (*Authorizer)(nil)

// NewAuthorizer builds the Authorizer from the static policy.
func NewAuthorizer(logger log.Logger, metricsHandler metrics.Handler) *Authorizer {
	a := &Authorizer{
		logger:          logger,
		metrics:         metricsHandler,
		allowed:         make(map[Identity]map[string]struct{}, len(allowList)),
		known:           make(map[string]struct{}),
		namespaceScoped: make(map[string]bool),
	}
	for id, apis := range allowList {
		set := make(map[string]struct{}, len(apis))
		for _, api := range apis {
			set[api] = struct{}{}
			a.known[api] = struct{}{}
			a.namespaceScoped[api] = requestHasNamespace(api)
		}
		a.allowed[id] = set
	}
	for _, api := range deniedByDefault {
		a.known[api] = struct{}{}
	}
	return a
}

// Authorize never panics and never returns an error: any failure is a deny.
func (a *Authorizer) Authorize(
	ctx context.Context,
	claims *authorization.Claims,
	target *authorization.CallTarget,
) (result authorization.Result, err error) {
	var api, namespace string
	if target != nil {
		api, namespace = target.APIName, target.Namespace
	}
	id := identityOf(claims)
	defer func() {
		if r := recover(); r != nil {
			result, err = deny, nil
			a.recordDeny(ctx, id, api, namespace, ReasonInternal, fmt.Sprint(r))
		}
	}()

	if reason := a.decide(id, api, namespace); reason != "" {
		a.recordDeny(ctx, id, api, namespace, reason, "")
		return deny, nil
	}
	if id == Worker && api == respondWorkflowTaskCompleted {
		if d := inspectCommands(target.Request); d != nil {
			a.recordCommandDeny(ctx, id, api, namespace, d)
			return deny, nil
		}
	}
	return allow, nil
}

// decide returns "" to allow, or the deny reason.
func (a *Authorizer) decide(id Identity, api, namespace string) string {
	switch {
	case api == healthCheck:
		return ""
	case id == "":
		return ReasonNoIdentity
	case id == System:
		return ""
	}
	if _, ok := a.allowed[id][api]; !ok {
		return ReasonAPI
	}
	if namespace == Namespace {
		return ""
	}
	if namespace == "" && !a.namespaceScoped[api] {
		return ""
	}
	return ReasonNamespace
}

func identityOf(claims *authorization.Claims) Identity {
	if claims == nil {
		return ""
	}
	id, _ := identityFromCN(claims.Subject)
	return id
}

func (a *Authorizer) recordDeny(ctx context.Context, id Identity, api, namespace, reason, detail string) {
	// Logging and metrics must not turn a deny into a panic either.
	defer func() { _ = recover() }()

	tags := a.denyTags(ctx, id, api, namespace, reason, commandNone)
	if detail != "" {
		tags = append(tags, tag.NewStringTag("detail", detail))
	}
	a.logger.Warn("Temporal API call denied", tags...)
}

// recordCommandDeny logs a denied workflow task response at error level with
// alert=true: the task retries until someone acts, so each one must page. The
// workflow ids go to the log line only; they hold the tenant and are unbounded.
//
// TODO(TODO-004): no alert rule is installed yet; install the
// nava_authz_denied{command!="none"} rule from docs/temporal-server.md.
func (a *Authorizer) recordCommandDeny(ctx context.Context, id Identity, api, namespace string, d *commandDenial) {
	defer func() { _ = recover() }()

	tags := a.denyTags(ctx, id, api, namespace, d.reason, d.command)
	tags = append(tags,
		tag.NewStringTag("alert", "true"),
		tag.NewStringTag("workflow_id", truncate(d.workflowID, maxLoggedWorkflowID)),
		tag.NewStringTag("command_index", strconv.Itoa(d.index)),
	)
	if d.target != "" {
		tags = append(tags, tag.NewStringTag("target_workflow_id", truncate(d.target, maxLoggedWorkflowID)))
	}
	if d.detail != "" {
		tags = append(tags, tag.NewStringTag("detail", d.detail))
	}
	a.logger.Error("Temporal workflow command denied", tags...)
}

// denyTags increments DeniedMetric and returns the log tags common to every deny.
func (a *Authorizer) denyTags(ctx context.Context, id Identity, api, namespace, reason, command string) []tag.Tag {
	identityTag := string(id)
	if identityTag == "" {
		identityTag = "none"
	}
	apiTag := api
	if _, ok := a.known[api]; !ok {
		apiTag = "other"
	}
	a.metrics.WithTags(
		metrics.StringTag("identity", identityTag),
		metrics.StringTag("api", apiTag),
		metrics.StringTag("namespace", namespaceTag(namespace)),
		metrics.StringTag("reason", reason),
		metrics.StringTag("command", command),
	).Counter(DeniedMetric).Record(1)

	tags := []tag.Tag{
		tag.NewStringTag("identity", identityTag),
		tag.NewStringTag("api", api),
		tag.NewStringTag("namespace", truncate(namespace, maxLoggedNamespace)),
		tag.NewStringTag("reason", reason),
		tag.NewStringTag("command", command),
	}
	if cert := authorization.PeerCert(authorization.TLSInfoFromContext(ctx)); cert != nil {
		tags = append(tags,
			tag.NewStringTag("cert_subject", cert.Subject.String()),
			tag.NewStringTag("cert_serial", cert.SerialNumber.String()),
			tag.NewStringTag("cert_issuer", cert.Issuer.String()),
		)
	} else {
		tags = append(tags, tag.NewStringTag("cert_subject", "none"))
	}
	return tags
}

// namespaceTag keeps the metric's cardinality bounded: the namespace is
// client-supplied and not yet validated when the Authorizer runs.
func namespaceTag(namespace string) string {
	switch namespace {
	case "":
		return "none"
	case Namespace, string(System):
		return namespace
	default:
		return "other"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// requestHasNamespace reports whether the API's request message has a
// `namespace` field, from the registered proto descriptors. An API it cannot
// resolve counts as namespace-scoped, so it must target Namespace (fail closed).
func requestHasNamespace(api string) bool {
	parts := strings.Split(strings.TrimPrefix(api, "/"), "/")
	if len(parts) != 2 {
		return true
	}
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(parts[0]))
	if err != nil {
		return true
	}
	svc, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return true
	}
	m := svc.Methods().ByName(protoreflect.Name(parts[1]))
	if m == nil {
		return true
	}
	return m.Input().Fields().ByName("namespace") != nil
}
