// Package roles is the db-roles job: run once with the bootstrap superuser,
// it converges the cluster's roles, their memberships, the databases, the
// service databases' grants and the infrastructure exceptions to
// db/grants.yaml, then exits. Every role is
// NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS (ADR-001 A.3), so the
// migrator cannot create roles and this job does it instead. A second run
// changes nothing, and a cluster with a second superuser fails the run before
// it changes anything.
package roles

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/scram"
)

// PasswordFunc returns the password for a LOGIN role, or false when none is set.
type PasswordFunc func(role string) (string, bool)

// PasswordEnv is the environment variable that holds role's password, e.g.
// DBROLES_PASSWORD_NAVA_APP.
func PasswordEnv(role string) string { return "DBROLES_PASSWORD_" + strings.ToUpper(role) }

// Result counts what one run did. A run on a converged cluster has every
// count zero.
type Result struct {
	Created  int // roles and databases
	Changed  int // attributes, passwords, owners, the lookup function
	Revoked  int // memberships, service-database grants and exception grants
	Existing int // roles and databases already present
}

// Total is the number of changes.
func (r Result) Total() int { return r.Created + r.Changed + r.Revoked }

// get_auth is the only infrastructure exception db-roles knows how to build.
const getAuthException = "pgbouncer_get_auth"

// Run converges the cluster. cfg is the bootstrap superuser's connection to
// any database; Run connects to the platform database itself for the
// exception objects.
func Run(ctx context.Context, cfg *pgx.ConnConfig, m *grants.Manifest, passwords PasswordFunc, log *slog.Logger) (Result, error) {
	var res Result
	if _, ok := m.InfrastructureExceptions[getAuthException]; !ok || len(m.InfrastructureExceptions) != 1 {
		return res, fmt.Errorf("db-roles builds exactly one infrastructure exception, %s", getAuthException)
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return res, err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	var me string
	var super bool
	if err := conn.QueryRow(ctx, "SELECT current_user::text, rolsuper FROM pg_catalog.pg_roles WHERE rolname = current_user").Scan(&me, &super); err != nil {
		return res, err
	}
	if !super {
		return res, fmt.Errorf("db-roles must run as the bootstrap superuser, not %q", me)
	}
	if _, ok := m.Roles[me]; ok {
		return res, fmt.Errorf("the manifest lists the bootstrap superuser %q as a role", me)
	}
	// One db-roles at a time.
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtext('nava.db-roles'))"); err != nil {
		return res, err
	}
	// Before any change: a cluster that fails this is left as it was.
	if err := onlySuperuser(ctx, conn, me, m.RoleNames()); err != nil {
		return res, err
	}

	for _, name := range m.RoleNames() {
		if err := ensureRole(ctx, conn, name, m.Roles[name], passwords, log, &res); err != nil {
			return res, fmt.Errorf("role %s: %w", name, err)
		}
	}
	if err := convergeMemberships(ctx, conn, m, log, &res); err != nil {
		return res, err
	}
	for _, name := range m.DatabaseNames() {
		created, err := ensureDatabase(ctx, conn, name, m.Databases[name].Owner, log, &res)
		if err != nil {
			return res, fmt.Errorf("database %s: %w", name, err)
		}
		if name == m.PlatformDatabase {
			// Its grants are the migrator's (grants.Apply). Until migrate runs,
			// PostgreSQL's default would give PUBLIC CONNECT and TEMPORARY on
			// it while keycloak and temporal start, so a new one starts with
			// no PUBLIC grants at all.
			if created {
				if err := revokePublic(ctx, conn, name, log, &res); err != nil {
					return res, fmt.Errorf("database %s: %w", name, err)
				}
			}
			continue
		}
		if err := ownerOnlyDatabase(ctx, conn, name, log, &res); err != nil {
			return res, fmt.Errorf("database %s: %w", name, err)
		}
	}

	pcfg := cfg.Copy()
	pcfg.Database = m.PlatformDatabase
	pconn, err := pgx.ConnectConfig(ctx, pcfg)
	if err != nil {
		return res, err
	}
	defer func() { _ = pconn.Close(context.WithoutCancel(ctx)) }()
	if err := ensureGetAuth(ctx, pconn, m, log, &res); err != nil {
		return res, fmt.Errorf("%s: %w", getAuthException, err)
	}
	log.InfoContext(ctx, "db-roles done",
		slog.Int("created", res.Created), slog.Int("changed", res.Changed),
		slog.Int("revoked", res.Revoked), slog.Int("existing", res.Existing))
	return res, nil
}

func ensureRole(ctx context.Context, conn *pgx.Conn, name string, r grants.Role, passwords PasswordFunc, log *slog.Logger, res *Result) error {
	login := r.HasAttribute(grants.AttrLogin)
	repl := r.HasAttribute(grants.AttrReplication)
	var pw string
	if login {
		var ok bool
		if pw, ok = passwords(name); !ok || pw == "" {
			return fmt.Errorf("LOGIN role has no password (set %s)", PasswordEnv(name))
		}
		if err := scram.CheckPassword(pw); err != nil {
			return err
		}
	}

	var super, createdb, createrole, bypass, canLogin, canRepl bool
	var stored *string
	created := false
	err := conn.QueryRow(ctx, `
		SELECT rolsuper, rolcreatedb, rolcreaterole, rolbypassrls, rolcanlogin, rolreplication, rolpassword
		FROM pg_catalog.pg_authid WHERE rolname = $1`, name).
		Scan(&super, &createdb, &createrole, &bypass, &canLogin, &canRepl, &stored)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := conn.Exec(ctx, "CREATE ROLE "+ident(name)+" NOLOGIN"); err != nil {
			return err
		}
		log.InfoContext(ctx, "created role", slog.String("role", name))
		res.Created++
		created = true
	case err != nil:
		return err
	default:
		res.Existing++
		if !super && !createdb && !createrole && !bypass && canLogin == login && canRepl == repl &&
			(!login || (stored != nil && scram.Matches(*stored, pw))) {
			return nil
		}
	}

	stmt := fmt.Sprintf("ALTER ROLE %s NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS %s %s",
		ident(name), onOff(login, "LOGIN"), onOff(repl, "REPLICATION"))
	if login && (stored == nil || !scram.Matches(*stored, pw)) {
		verifier, err := scram.Encode(pw)
		if err != nil {
			return err
		}
		// A verifier, not the password: nothing readable reaches the server
		// or its statement log.
		stmt += " PASSWORD " + literal(verifier)
	}
	if !login {
		stmt += " PASSWORD NULL"
	}
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return err
	}
	if !created {
		log.WarnContext(ctx, "converged role attributes or password", slog.String("role", name))
		res.Changed++
	}
	return nil
}

// convergeMemberships revokes every membership that involves a manifest
// role, as member or as granted role. The manifest lists none: ADR-001 I3
// allows no runtime role to be a member of any role, pg_* included, and
// D28 grants the migrator to no one.
func convergeMemberships(ctx context.Context, conn *pgx.Conn, m *grants.Manifest, log *slog.Logger, res *Result) error {
	rows, err := conn.Query(ctx, `
		SELECT r.rolname::text, u.rolname::text, g.rolname::text
		FROM pg_catalog.pg_auth_members am
		JOIN pg_catalog.pg_roles r ON r.oid = am.roleid
		JOIN pg_catalog.pg_roles u ON u.oid = am.member
		JOIN pg_catalog.pg_roles g ON g.oid = am.grantor
		WHERE r.rolname = ANY ($1) OR u.rolname = ANY ($1)
		ORDER BY 1, 2, 3`, m.RoleNames())
	if err != nil {
		return err
	}
	type membership struct{ Role, Member, Grantor string }
	ms, err := pgx.CollectRows(rows, pgx.RowToStructByPos[membership])
	if err != nil {
		return err
	}
	for _, x := range ms {
		if _, err := conn.Exec(ctx, fmt.Sprintf("REVOKE %s FROM %s GRANTED BY %s CASCADE",
			ident(x.Role), ident(x.Member), ident(x.Grantor))); err != nil {
			return fmt.Errorf("revoke %s from %s: %w", x.Role, x.Member, err)
		}
		log.WarnContext(ctx, "revoked role membership not in manifest",
			slog.String("role", x.Role), slog.String("member", x.Member), slog.String("grantor", x.Grantor))
		res.Revoked++
	}
	return nil
}

// onlySuperuser fails unless me is the cluster's only superuser once the
// manifest roles are converged: ensureRole makes those NOSUPERUSER, so only a
// superuser outside the manifest fails the run. The superuser rules in
// db/pg_hba.conf match the bootstrap superuser by name, so a second superuser
// would log in over TCP through the catch-all.
func onlySuperuser(ctx context.Context, conn *pgx.Conn, me string, managed []string) error {
	var others []string
	if err := conn.QueryRow(ctx, `
		SELECT coalesce(array_agg(rolname::text ORDER BY rolname), '{}')
		FROM pg_catalog.pg_roles WHERE rolsuper AND rolname <> $1 AND rolname <> ALL ($2)`, me, managed).Scan(&others); err != nil {
		return err
	}
	if len(others) > 0 {
		return fmt.Errorf("%q must be the only superuser; also SUPERUSER: %s", me, strings.Join(others, ", "))
	}
	return nil
}

// ensureDatabase reports whether it created the database.
func ensureDatabase(ctx context.Context, conn *pgx.Conn, name, owner string, log *slog.Logger, res *Result) (bool, error) {
	var cur string
	err := conn.QueryRow(ctx, `
		SELECT pg_catalog.pg_get_userbyid(datdba)::text FROM pg_catalog.pg_database WHERE datname = $1`, name).Scan(&cur)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s OWNER %s", ident(name), ident(owner))); err != nil {
			return false, err
		}
		log.InfoContext(ctx, "created database", slog.String("database", name), slog.String("owner", owner))
		res.Created++
		return true, nil
	case err != nil:
		return false, err
	}
	res.Existing++
	if cur == owner {
		return false, nil
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", ident(name), ident(owner))); err != nil {
		return false, err
	}
	log.WarnContext(ctx, "changed database owner", slog.String("database", name),
		slog.String("from", cur), slog.String("to", owner))
	res.Changed++
	return false, nil
}

// revokePublic removes PostgreSQL's default CONNECT and TEMPORARY for PUBLIC
// from a database db-roles has just created.
func revokePublic(ctx context.Context, conn *pgx.Conn, name string, log *slog.Logger, res *Result) error {
	if _, err := conn.Exec(ctx, "REVOKE ALL ON DATABASE "+ident(name)+" FROM PUBLIC"); err != nil {
		return err
	}
	log.InfoContext(ctx, "revoked PUBLIC grants on new database", slog.String("database", name))
	res.Revoked++
	return nil
}

// ownerOnlyDatabase revokes every privilege on a service database (keycloak,
// temporal, temporal_visibility) held by anyone but its owner, PUBLIC
// included. PostgreSQL's default gives PUBLIC CONNECT and TEMPORARY on a new
// database, which would let every runtime role connect to Keycloak's and
// Temporal's data (ADR-001 A.3: "No CONNECT on nava, and the reverse").
func ownerOnlyDatabase(ctx context.Context, conn *pgx.Conn, name string, log *slog.Logger, res *Result) error {
	// A superuser's REVOKE acts as the owner, so a grant made by anyone else
	// (only possible through a grant option) would survive it: read twice.
	type acl struct{ Grantee, Priv string }
	read := func() ([]acl, error) {
		rows, err := conn.Query(ctx, `
			SELECT CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_catalog.pg_get_userbyid(a.grantee)::text END,
			       a.privilege_type
			FROM pg_catalog.pg_database d,
			     pg_catalog.aclexplode(coalesce(d.datacl, pg_catalog.acldefault('d', d.datdba))) a
			WHERE d.datname = $1 AND a.grantee <> d.datdba
			ORDER BY 1, 2`, name)
		if err != nil {
			return nil, err
		}
		return pgx.CollectRows(rows, pgx.RowToStructByPos[acl])
	}
	acls, err := read()
	if err != nil {
		return err
	}
	for _, a := range acls {
		grantee := "PUBLIC"
		if a.Grantee != "PUBLIC" {
			grantee = ident(a.Grantee)
		}
		// a.Priv comes from aclexplode: CONNECT, TEMPORARY or CREATE.
		if _, err := conn.Exec(ctx, fmt.Sprintf("REVOKE %s ON DATABASE %s FROM %s CASCADE",
			a.Priv, ident(name), grantee)); err != nil {
			return fmt.Errorf("revoke %s from %s: %w", a.Priv, a.Grantee, err)
		}
		log.WarnContext(ctx, "revoked database grant not in manifest", slog.String("database", name),
			slog.String("grantee", a.Grantee), slog.String("privilege", a.Priv))
		res.Revoked++
	}
	if acls, err = read(); err != nil {
		return err
	} else if len(acls) > 0 {
		return fmt.Errorf("%s on database %s survived its revoke; its grantor is not the owner", acls[0].Priv, name)
	}
	return nil
}

// ensureGetAuth builds PgBouncer's auth_query function and keeps its grants
// to exactly the exception's granted_to. The function returns a role's
// verifier only for an allow-list taken from pgbouncer_login in the manifest,
// and never for a superuser, BYPASSRLS or REPLICATION role, so the migrator,
// the backup role and the bootstrap superuser cannot log in through the
// pooler. The allow-list is a literal in the function body rather than
// membership of a group role, because ADR-001 I3 allows runtime roles no
// memberships at all.
func ensureGetAuth(ctx context.Context, conn *pgx.Conn, m *grants.Manifest, log *slog.Logger, res *Result) error {
	ex := m.InfrastructureExceptions[getAuthException]
	if ex.Function != ex.Schema+".get_auth(text)" || !ex.SecurityDefiner {
		return fmt.Errorf("expected a security_definer function %s.get_auth(text), got %q", ex.Schema, ex.Function)
	}
	allow := m.PgBouncerLogins()
	if allow == nil {
		allow = []string{}
	}
	var arr string
	if err := conn.QueryRow(ctx, "SELECT $1::text[]::text", allow).Scan(&arr); err != nil {
		return err
	}
	body := "SELECT usename, passwd FROM pg_catalog.pg_shadow" +
		" WHERE usename = p_usename AND usename = ANY (" + literal(arr) + "::name[])" +
		" AND NOT usesuper AND NOT usebypassrls AND NOT userepl"

	schemaExists, err := superuserOwned(ctx, conn, ex.Schema, ex.Function)
	if err != nil {
		return err
	}
	if !schemaExists {
		// Not IF NOT EXISTS: a schema created since the check fails the run
		// instead of being adopted.
		if _, err := conn.Exec(ctx, "CREATE SCHEMA "+ident(ex.Schema)); err != nil {
			return err
		}
	}
	var src, cfg *string
	var secdef bool
	err = conn.QueryRow(ctx, `
		SELECT p.prosrc, p.prosecdef, array_to_string(p.proconfig, ',')
		FROM pg_catalog.pg_proc p WHERE p.oid = pg_catalog.to_regprocedure($1)`, ex.Function).Scan(&src, &secdef, &cfg)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) || src == nil || *src != body || !secdef || cfg == nil || *cfg != "search_path=pg_catalog" {
		stmt := fmt.Sprintf(`CREATE OR REPLACE FUNCTION %s.get_auth(p_usename text)
			RETURNS TABLE (usename name, passwd text)
			LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog
			AS %s`, ident(ex.Schema), literal(body))
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
		log.InfoContext(ctx, "wrote PgBouncer lookup function", slog.String("function", ex.Function),
			slog.Any("allow", allow))
		res.Changed++
	}

	// Grants on the schema and the function: exactly granted_to, never PUBLIC.
	targets := []struct{ on, kind, priv string }{
		{"SCHEMA " + ident(ex.Schema), "schema", "USAGE"},
		{"FUNCTION " + ex.Function, "function", "EXECUTE"},
	}
	for _, t := range targets {
		var q string
		if t.kind == "schema" {
			q = `SELECT CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_catalog.pg_get_userbyid(a.grantee)::text END, a.privilege_type, a.is_grantable
				FROM pg_catalog.pg_namespace n, pg_catalog.aclexplode(coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))) a
				WHERE n.nspname = $1 AND a.grantee <> n.nspowner`
		} else {
			q = `SELECT CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_catalog.pg_get_userbyid(a.grantee)::text END, a.privilege_type, a.is_grantable
				FROM pg_catalog.pg_proc p, pg_catalog.aclexplode(coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
				WHERE p.oid = pg_catalog.to_regprocedure($1) AND a.grantee <> p.proowner`
		}
		arg := ex.Schema
		if t.kind == "function" {
			arg = ex.Function
		}
		rows, err := conn.Query(ctx, q, arg)
		if err != nil {
			return err
		}
		type acl struct {
			Grantee, Priv string
			Grantable     bool
		}
		acls, err := pgx.CollectRows(rows, pgx.RowToStructByPos[acl])
		if err != nil {
			return err
		}
		have := map[string]bool{}
		for _, a := range acls {
			if a.Priv == t.priv && !a.Grantable && slices.Contains(ex.GrantedTo, a.Grantee) {
				have[a.Grantee] = true
				continue
			}
			grantee := "PUBLIC"
			if a.Grantee != "PUBLIC" {
				grantee = ident(a.Grantee)
			}
			if _, err := conn.Exec(ctx, fmt.Sprintf("REVOKE ALL ON %s FROM %s CASCADE", t.on, grantee)); err != nil {
				return err
			}
			log.WarnContext(ctx, "revoked grant not in manifest", slog.String("object", t.on),
				slog.String("grantee", a.Grantee), slog.String("privilege", a.Priv))
			res.Revoked++
		}
		for _, r := range ex.GrantedTo {
			if have[r] {
				continue
			}
			if _, err := conn.Exec(ctx, fmt.Sprintf("GRANT %s ON %s TO %s", t.priv, t.on, ident(r))); err != nil {
				return err
			}
			log.InfoContext(ctx, "granted", slog.String("object", t.on), slog.String("grantee", r), slog.String("privilege", t.priv))
			res.Changed++
		}
	}
	return nil
}

// superuserOwned fails unless the schema and the function, where they exist,
// are owned by a superuser. A role with CREATE on the platform database, the
// migrator as its owner, could create the schema before db-roles and then
// redefine get_auth; CREATE SCHEMA IF NOT EXISTS and CREATE OR REPLACE keep
// the existing owner, so db-roles checks before either and repairs nothing
// (ADR-001 I3, TICK-107). It reports whether the schema exists.
func superuserOwned(ctx context.Context, conn *pgx.Conn, schema, function string) (bool, error) {
	type owned struct {
		Kind, Owner string
		Super       bool
	}
	rows, err := conn.Query(ctx, `
		SELECT 'schema', r.rolname::text, r.rolsuper
		FROM pg_catalog.pg_namespace n JOIN pg_catalog.pg_roles r ON r.oid = n.nspowner
		WHERE n.nspname = $1
		UNION ALL
		SELECT 'function', r.rolname::text, r.rolsuper
		FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
		WHERE p.oid = pg_catalog.to_regprocedure($2)`, schema, function)
	if err != nil {
		return false, err
	}
	objs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[owned])
	if err != nil {
		return false, err
	}
	for _, o := range objs {
		if o.Super {
			continue
		}
		name := schema
		if o.Kind == "function" {
			name = function
		}
		return false, fmt.Errorf("%s %s is owned by %q, which is not a superuser; drop it or change its owner by hand, db-roles will not repair it",
			o.Kind, name, o.Owner)
	}
	// The function lives in the schema, so any row means the schema exists.
	return len(objs) > 0, nil
}

func onOff(on bool, attr string) string {
	if on {
		return attr
	}
	return "NO" + attr
}

func ident(s string) string { return pgx.Identifier{s}.Sanitize() }

// literal quotes s as a standard SQL string literal.
func literal(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
