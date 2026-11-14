// Command server is the ADR-007 S4 scratch server. It keeps the order of the
// internal/server chain (authn 401 -> client gate and origin check 403 ->
// limiter 429 -> handler) and adds the D3 CORS handler either outermost
// (-cors=outer, what the ADR says) or innermost (-cors=inner, a negative
// control that shows why the order matters).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

type who struct {
	tenant   uuid.UUID
	clientID string
}

type ctxKey struct{}

//nolint:gochecknoglobals // scratch: static stand-ins for verified tokens; the real chain resolves (iss, organization.id, azp).
var tokens = map[string]who{
	"tok-ok":      {uuid.MustParse("00000000-0000-0000-0000-000000000001"), "host-app-1"},
	"tok-limited": {uuid.MustParse("00000000-0000-0000-0000-000000000002"), "host-app-1"},
	"tok-off":     {uuid.MustParse("00000000-0000-0000-0000-000000000003"), "host-app-7"}, // disabled row
}

func problem(w http.ResponseWriter, status int, code string, hdr ...string) {
	w.Header().Set("Content-Type", "application/problem+json")
	for i := 0; i+1 < len(hdr); i += 2 {
		w.Header().Set(hdr[i], hdr[i+1])
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "code": code})
}

// cors is the D3 handler, exactly.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", o)
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Expose-Headers", "ETag, Location, Retry-After")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, If-Match, Nava-SDK")
				h.Set("Access-Control-Max-Age", "7200")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8087", "listen address")
	dsn := flag.String("dsn", "", "nava_app DSN through PgBouncer")
	mode := flag.String("cors", "outer", "outer | inner")
	flag.Parse()
	pool, err := pgxpool.New(context.Background(), *dsn)
	if err != nil {
		log.Fatal(err)
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"v1"`)
		if r.Method == http.MethodPost {
			w.Header().Set("Location", "/api/v1/records/loan/1")
			w.WriteHeader(http.StatusCreated)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "method": r.Method})
	})
	limiter := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.Header.Get("Authorization"), "tok-limited") {
				problem(w, 429, "rate_limited", "Retry-After", "7")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	// gate is the D2 lookup plus the D3 origin check, inside InTenantTx.
	gate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.Context().Value(ctxKey{}).(who)
			var origins []string
			var state string
			err := tenancy.InTenantTx(r.Context(), pool, p.tenant, func(tx pgx.Tx) error {
				return tx.QueryRow(r.Context(),
					`SELECT allowed_origins, state FROM embedding_client WHERE tenant_id = $1 AND client_id = $2`,
					p.tenant, p.clientID).Scan(&origins, &state)
			})
			if err != nil || state != "enabled" {
				problem(w, 403, "client_not_enabled")
				return
			}
			if o := r.Header.Get("Origin"); o != "" {
				ok := false
				for _, a := range origins {
					ok = ok || a == o
				}
				if !ok {
					problem(w, 403, "origin_not_allowed")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
	authn := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t, ok := tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
			if !ok {
				problem(w, 401, "unauthenticated", "WWW-Authenticate", "Bearer")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, t)))
		})
	}
	var h http.Handler
	if *mode == "outer" {
		h = cors(authn(gate(limiter(final))))
	} else {
		// Negative control: CORS wraps only the handler, so the preflight and
		// every 401/403/429 leave without CORS headers.
		h = authn(gate(limiter(cors(final))))
	}
	log.Printf("cors=%s listening on %s", *mode, *addr)
	srv := &http.Server{Addr: *addr, Handler: h, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
