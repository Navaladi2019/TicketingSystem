package compose

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The connection budget (docs/budgets.md "Connection sizing", ADR-005 A5,
// AR-TICK-653-04): every server connection the compose stack can open to its
// one PostgreSQL, counted at its configured ceiling, must fit in
// max_connections minus superuser_reserved_connections.
//
// Through PgBouncer the ceiling is one server pool per (database entry, role)
// pair the stack logs in with, sized as PgBouncer sizes it: the role's
// [users] pool_size, else the entry's pool_size, else default_pool_size. The
// auth_user's lookup pool is counted once per entry. Direct logins are
// Keycloak's pool, Temporal's SQL pools (one per store per service, all
// services in one process) and the two one-shot schema jobs.

const (
	budgetCompose   = "../../compose.yaml"
	budgetComposeCI = "compose.ci.yaml"
	pgbouncerINI    = "pgbouncer/pgbouncer.ini"
	temporalConfig  = "temporal/config/compose.yaml"
	allocatorGo     = "../../cmd/nava-core/allocator.go"
	workerMainGo    = "../../cmd/nava-worker/main.go"
	// PostgreSQL's default superuser_reserved_connections; db-roles, the
	// only superuser login, uses it.
	defaultSuperuserReserved = 3
)

type pgbouncerConfig struct {
	DefaultPoolSize int
	AuthUser        string
	// DBPoolSize is each [databases] entry's pool_size, -1 when unset.
	DBPoolSize map[string]int
	// UserPoolSize is each [users] entry's pool_size.
	UserPoolSize map[string]int
}

var iniKV = regexp.MustCompile(`(\w+)\s*=\s*(\S+)`)

func parsePgBouncerINI(t *testing.T, file string) pgbouncerConfig {
	t.Helper()
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c := pgbouncerConfig{DefaultPoolSize: 20, DBPoolSize: map[string]int{}, UserPoolSize: map[string]int{}}
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.Trim(line, "[]")
			continue
		}
		name, val, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("%s: %q is not name = value", file, line)
		}
		name, val = strings.TrimSpace(name), strings.TrimSpace(val)
		poolSize := func() int {
			for _, m := range iniKV.FindAllStringSubmatch(val, -1) {
				if m[1] == "pool_size" {
					n, err := strconv.Atoi(m[2])
					if err != nil {
						t.Fatalf("%s: %s pool_size %q", file, name, m[2])
					}
					return n
				}
			}
			return -1
		}
		switch section {
		case "databases":
			c.DBPoolSize[name] = poolSize()
		case "users":
			if n := poolSize(); n >= 0 {
				c.UserPoolSize[name] = n
			}
		case "pgbouncer":
			switch name {
			case "default_pool_size":
				n, err := strconv.Atoi(val)
				if err != nil {
					t.Fatalf("%s: default_pool_size %q", file, val)
				}
				c.DefaultPoolSize = n
			case "auth_user":
				c.AuthUser = val
			case "reserve_pool_size":
				// A reserve pool adds server connections on top of every
				// pool; the budget below does not count them.
				if val != "0" {
					t.Fatalf("%s: reserve_pool_size = %s, which this budget does not count", file, val)
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return c
}

// serverPool is the server pool size PgBouncer gives role on entry db.
func (c pgbouncerConfig) serverPool(db, role string) int {
	if n, ok := c.UserPoolSize[role]; ok {
		return n
	}
	if n := c.DBPoolSize[db]; n >= 0 {
		return n
	}
	return c.DefaultPoolSize
}

type composeService struct {
	Command     []string       `yaml:"command"`
	Environment map[string]any `yaml:"environment"`
	Deploy      struct {
		Replicas *int `yaml:"replicas"`
	} `yaml:"deploy"`
}

func loadComposeServices(t *testing.T, file string) map[string]composeService {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Services map[string]composeService `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Services
}

var envDefault = regexp.MustCompile(`\$\{[A-Za-z_]+:-([^}]*)\}`)

func envString(v any) string { return envDefault.ReplaceAllString(fmt.Sprint(v), "$1") }

// postgresSetting reads `-c name=value` from the postgres command, or def.
func postgresSetting(t *testing.T, cmd []string, name string, def int) int {
	t.Helper()
	for i, a := range cmd {
		if a != "-c" || i+1 >= len(cmd) {
			continue
		}
		k, v, _ := strings.Cut(cmd[i+1], "=")
		if k == name {
			n, err := strconv.Atoi(v)
			if err != nil {
				t.Fatalf("postgres -c %s=%q", name, v)
			}
			return n
		}
	}
	return def
}

func replicas(s composeService) int {
	if s.Deploy.Replicas != nil {
		return *s.Deploy.Replicas
	}
	return 1
}

// temporalCeiling is maxConns summed over the datastores, times the services
// one Temporal process runs (each opens its own pool per store).
func temporalCeiling(t *testing.T) (perService, services int) {
	t.Helper()
	b, err := os.ReadFile(temporalConfig)
	if err != nil {
		t.Fatal(err)
	}
	inServices := false
	for _, line := range strings.Split(string(b), "\n") {
		if m := regexp.MustCompile(`^\s+maxConns:\s*(\d+)`).FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			perService += n
		}
		switch {
		case line == "services:":
			inServices = true
		case inServices && regexp.MustCompile(`^\S`).MatchString(line):
			inServices = false
		case inServices && regexp.MustCompile(`^  [a-zA-Z]+:\s*$`).MatchString(line):
			services++
		}
	}
	if perService == 0 || services == 0 {
		t.Fatalf("%s: found maxConns %d over %d services", temporalConfig, perService, services)
	}
	return perService, services
}

func allocatorMaxConns(t *testing.T) int {
	t.Helper()
	b, err := os.ReadFile(allocatorGo)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^const allocatorMaxConns = (\d+)$`).FindSubmatch(b)
	if m == nil {
		t.Fatalf("%s: no `const allocatorMaxConns = N`", allocatorGo)
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// workerConst reads `name = N` from the const block of cmd/nava-worker/main.go.
func workerConst(t *testing.T, name string) int {
	t.Helper()
	b, err := os.ReadFile(workerMainGo)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^\s+` + name + `\s*= (\d+)$`).FindSubmatch(b)
	if m == nil {
		t.Fatalf("%s: no `%s = N` constant", workerMainGo, name)
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// healthProbeConns is the worker's readiness probe (health.Postgres): one
// connection of its own on the same role.
const healthProbeConns = 1

// TestWorkerPoolsFitPgBouncer (TICK-712 review): through PgBouncer the
// worker's pools share nava_worker's server pool. The main pool, the record
// pipeline's own pool and the readiness probe must all fit in it, or a burst
// of writes holds every server connection in an open transaction while their
// audit transactions wait for one.
func TestWorkerPoolsFitPgBouncer(t *testing.T) {
	t.Parallel()
	pgb := parsePgBouncerINI(t, pgbouncerINI)
	worker, pipe := workerConst(t, "workerMaxConns"), workerConst(t, "pipelineMaxConns")
	got := pgb.serverPool("nava", "nava_worker")
	if need := worker + pipe + healthProbeConns; need > got {
		t.Errorf("worker pool %d + pipeline pool %d + probe %d = %d > PgBouncer nava/nava_worker pool_size %d",
			worker, pipe, healthProbeConns, need, got)
	}
	if _, ok := pgb.UserPoolSize["nava_worker"]; !ok {
		t.Error("[users] nava_worker has no pool_size: the worker would fall back to default_pool_size")
	}
}

type budgetLine struct {
	What string
	N    int
}

// connectionBudget lists every ceiling and its total.
func connectionBudget(t *testing.T, svcs map[string]composeService, pgb pgbouncerConfig) ([]budgetLine, int) {
	t.Helper()
	var lines []budgetLine
	// PgBouncer pairs: every postgres:// URL in a service's environment that
	// points at the pooler. A pair is one server pool however many services
	// or replicas share it.
	pairs := map[[2]string]bool{}
	entries := map[string]bool{}
	for name, s := range svcs {
		for k, v := range s.Environment {
			u, err := url.Parse(envString(v))
			if err != nil || u.Scheme != "postgres" || u.Hostname() != "pgbouncer" {
				continue
			}
			db := strings.TrimPrefix(u.Path, "/")
			if _, ok := pgb.DBPoolSize[db]; !ok {
				t.Errorf("%s %s: PgBouncer has no [databases] entry %q", name, k, db)
			}
			pairs[[2]string{db, u.User.Username()}] = true
			entries[db] = true
		}
	}
	keys := make([][2]string, 0, len(pairs))
	for p := range pairs {
		keys = append(keys, p)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+"/"+keys[i][1] < keys[j][0]+"/"+keys[j][1] })
	for _, p := range keys {
		lines = append(lines, budgetLine{"pgbouncer " + p[0] + "/" + p[1], pgb.serverPool(p[0], p[1])})
	}
	for db := range entries {
		lines = append(lines, budgetLine{"pgbouncer " + db + "/" + pgb.AuthUser + " (auth_query)", pgb.serverPool(db, pgb.AuthUser)})
	}
	kc, err := strconv.Atoi(envString(svcs["keycloak"].Environment["KC_DB_POOL_MAX_SIZE"]))
	if err != nil {
		t.Fatalf("keycloak KC_DB_POOL_MAX_SIZE: %v", err)
	}
	lines = append(lines, budgetLine{"keycloak KC_DB_POOL_MAX_SIZE", kc * replicas(svcs["keycloak"])})
	per, n := temporalCeiling(t)
	lines = append(lines, budgetLine{fmt.Sprintf("temporal-server %d services x %d", n, per), per * n * replicas(svcs["temporal-server"])})
	lines = append(lines, budgetLine{"migrate (goose, one-shot)", 1}, budgetLine{"temporal-schema (one-shot)", 1})
	total := 0
	for _, l := range lines {
		total += l.N
	}
	return lines, total
}

// TestConnectionBudgetFits: the configured ceilings add up.
func TestConnectionBudgetFits(t *testing.T) {
	t.Parallel()
	svcs := loadComposeServices(t, budgetCompose)
	pgb := parsePgBouncerINI(t, pgbouncerINI)
	cmd := svcs["postgres"].Command
	maxConn := postgresSetting(t, cmd, "max_connections", 100)
	reserved := postgresSetting(t, cmd, "superuser_reserved_connections", defaultSuperuserReserved)
	lines, total := connectionBudget(t, svcs, pgb)
	for _, l := range lines {
		t.Logf("%4d  %s", l.N, l.What)
	}
	t.Logf("%4d  total; max_connections %d - reserved %d = %d", total, maxConn, reserved, maxConn-reserved)
	if total > maxConn-reserved {
		t.Errorf("connection ceilings total %d > max_connections %d - superuser_reserved_connections %d: "+
			"raise max_connections in compose.yaml and deploy/compose/compose.ci.yaml, and docs/budgets.md", total, maxConn, reserved)
	}
	// compose.ci.yaml replaces the postgres command; it must keep the setting.
	ci := loadComposeServices(t, budgetComposeCI)
	if got := postgresSetting(t, ci["postgres"].Command, "max_connections", 100); got != maxConn {
		t.Errorf("compose.ci.yaml postgres max_connections = %d, compose.yaml has %d", got, maxConn)
	}
}

// TestAllocatorPoolSizing (ADR-005 A5): PgBouncer's nava_alloc server pool is
// the allocator pool's width times the nava-core replicas, and nothing else
// widens it.
func TestAllocatorPoolSizing(t *testing.T) {
	t.Parallel()
	svcs := loadComposeServices(t, budgetCompose)
	pgb := parsePgBouncerINI(t, pgbouncerINI)
	want := allocatorMaxConns(t) * replicas(svcs["nava-core"])
	if got := pgb.serverPool("nava_alloc", "nava_app"); got != want {
		t.Errorf("PgBouncer nava_alloc/nava_app pool = %d, want allocatorMaxConns x nava-core replicas = %d", got, want)
	}
	if _, ok := pgb.UserPoolSize["nava_app"]; ok {
		t.Error("[users] nava_app pool_size would override nava_alloc's pool_size (PgBouncer takes the user's first)")
	}
}

// TestConnectionBudgetCountsPrecedence: the sizing rule is PgBouncer's
// (user, then entry, then default), so a [users] cap is what is counted.
func TestConnectionBudgetCountsPrecedence(t *testing.T) {
	t.Parallel()
	c := pgbouncerConfig{DefaultPoolSize: 20, DBPoolSize: map[string]int{"nava": -1, "nava_alloc": 4}, UserPoolSize: map[string]int{"r": 5}}
	for _, tc := range []struct {
		db, role string
		want     int
	}{{"nava", "x", 20}, {"nava_alloc", "x", 4}, {"nava", "r", 5}, {"nava_alloc", "r", 5}} {
		if got := c.serverPool(tc.db, tc.role); got != tc.want {
			t.Errorf("serverPool(%s, %s) = %d, want %d", tc.db, tc.role, got, tc.want)
		}
	}
}
