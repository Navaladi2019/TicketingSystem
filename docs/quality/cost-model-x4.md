# M-X4: infrastructure cost per agent seat (ADR-008 X4)

Scope: TICK-886, PLAN-009 P9-07 (`docs/delivery/plan-adr-008.md`, D-9-7). This is the **method** ADR-008 D9 asks for: infrastructure cost per agent seat per month at **50 and 500 seats**, from stated inputs. It sets **no target**. The X4 target is ADR-008 Q1, a business tradeoff for the human. It makes **no ratio or multiplier claim** (ADR-008 D4).

Recompute: `python -I docs/quality/cost-model-x4/model.py` prints every table under "Sizing", "Cost per month" and "Sensitivity" from the input rows. The script holds one constant per input row, named by the row's ID.

## Result

**Example prices** (table P), **HA** (A8 = 2). Infrastructure only: what is excluded is listed under "Not in the model".

| Seats | Low load (A1 × 0.5) | Expected load (A1) | High load (A1 × 2) |
|---|---|---|---|
| 50 | $9.84 | $9.86 | $9.90 |
| 500 | $1.40 | $2.22 | $3.87 |

$ per agent seat per month.

**Shown beside it, not compared with it.** Competitor **list prices** per user per month, from ADR-008 X4 and `docs/adr/competitive/pain-points.md`. A list price is what a customer pays for software, support, R&D and margin; the figures above are only what our infrastructure costs to run. They are not the same quantity, so no ratio between them is stated (D4).

| Register entry | List price per user per month | Status |
|---|---|---|
| CP-SF-04 Salesforce Service Cloud | $175 Enterprise / $350 Unlimited | Confirmed list price |
| CP-SN-01 ServiceNow ITSM | ~$90–150 per fulfiller | Third-party estimates, not list |
| CP-FW-04 Freshdesk | ~$55 Pro / ~$89 Enterprise | Confirmed list price |

**What moves the number** (from the tables below):

1. **The fixed floor at 50 seats.** At 50 seats load does not matter: the load tier stays at its 4 vCPU floor (A7) at every load. Compute and memory are $463.75 of $492.84 (94%), almost all of it always-on services (A9) and the floor. The per-seat cost at 50 seats is that floor divided by 50.
2. **HA (A8).** Running every service twice roughly doubles the cost: single instance is $5.18 at 50 seats and $1.15 at 500.
3. **The vCPU price (P1).** Compute is the largest line at every size (77% at 50 seats and at 500 expected). Doubling P1 moves 50 seats from $9.86 to $17.46.
4. **Load per seat at 500 seats** (A1 requests per seat, A4 cost of a request). These set the load tier: 8, 16 and 32 vCPU at low, expected and high. A4 from 1 to 4 creates moves 500 seats from $1.42 to $3.82.

Storage, backups, egress and the load balancer together are 7% or less of the total at every size (the most is 6.3%, at 50 seats and high load). Data volume does not move the number at these sizes.

## Inputs

### Deployment (D rows): what runs

Source: `compose.yaml` on `main`. The compose stack declares **no CPU or memory requests or limits**, so the size of each service is assumption A7 or A9, not a deployment fact.

| ID | Input | Value | Source |
|---|---|---|---|
| D1 | Load-scaled services | `nava-core` (API), `postgres` | `compose.yaml` |
| D2 | Always-on services | `keycloak`, `temporal-server` (4 Temporal services in one process), `temporal-ui`, `pgbouncer`, `nava-worker`, `nava-relay`, `nava-purge`, `nava-onboard`, `nava-groupsync`, `otel-collector`, `prometheus`, `openobserve` | `compose.yaml` |
| D3 | One-shot jobs, not costed | `db-roles`, `migrate`, `temporal-schema`, `temporal` (bootstrap) | `compose.yaml` |
| D4 | One PostgreSQL serves the platform, Temporal and Keycloak. Each extra `nava-core` replica adds 24 server connections; revisit beyond about 10 API replicas (AR-TICK-653-04) | 226 of 247 connections at 1 replica | `docs/budgets.md` "Connection sizing" |

At 500 seats and high load (100 req/s peak) the load tier is 32 vCPU. The model does not split it into replicas. If it ran as more than about 10 `nava-core` replicas, D4's revisit trigger fires; the model does not price that change.

### Measured (M rows)

| ID | Input | Value | Source |
|---|---|---|---|
| M1 | Throughput one host carried | **200 creates/s** (`gaps` type, open loop, 60 s), p95 33.2 ms, 0 errors, **not saturated** (the 10-connection API pool was the first limit) | `docs/quality/acl-perf-800k.md` "100/s open-loop create" (TICK-700) |
| M1 | Host for M1 | 16 vCPU, 19.4 GiB VM (WSL2 on an i7-1260P), `postgres:16.10`, 800k records in 400 tenants | `docs/quality/acl-perf-800k.md` |
| M2 | Database time of one create | 12.6 ms (four round trips) plus about 3 ms in the handler, at 100/s | same section |
| M3 | Latency budgets the sizing must keep | single read p95 ≤ 50 ms; list p95 ≤ 150 ms; create p95 ≤ 50 ms (met on M1's host) | `docs/budgets.md`; `docs/quality/acl-perf-800k.md` "Final verdict" |
| M4 | PgBouncer in the path | correctness only; no throughput measured | `docs/quality/pgbouncer-load.md` "Not tested" |

M1 is used as a **lower bound** on what 16 vCPU carry: the host was not saturated, and it also ran PostgreSQL. Treating it as a ceiling overstates the vCPU needed, so the error is towards a higher cost. M1's known gaps carry over: no PgBouncer, no HTTP transport, one host on a WSL2 virtual disk (`AR-S10-02`).

### Assumptions (A rows)

Each is an assumption, not a measurement. A1 is varied × 0.5 and × 2 for the low and high columns, and A10 scales with it.

| ID | Assumption | Value | Why this value |
|---|---|---|---|
| A1 | API requests per agent seat per active hour | **120** (low 60, high 240) | Two requests a minute: a list or read per ticket touched, plus updates. No usage data exists yet; replace with the first pilot's telemetry |
| A2 | Active hours per seat per month | 176 (22 days × 8 h) | One shift per seat |
| A3 | Peak factor (busiest minute / active-hour average) | 3 | All seats active together, plus bursts |
| A4 | Cost of one mixed request, in measured creates (M1) | 2 | Lists are slower than creates (p95 up to 146 ms against 33 ms); reads are cheaper. Varied 1 to 4 in the sensitivity table |
| A5 | Target CPU utilisation of the load tier at peak | 50% | Headroom for the latency budgets (M3) |
| A6 | Memory per load-tier vCPU | 2 GiB | M1's host had 1.2 GiB per vCPU with `shared_buffers = 2GB`; rounded up for PostgreSQL cache |
| A7 | Load-tier floor | 4 vCPU (`nava-core` 1, `postgres` 3) | Keeps PostgreSQL off a single core while Temporal and Keycloak share it (D4) |
| A8 | High availability | 2: every service runs twice; PostgreSQL as primary plus standby | Production shape. 1 = single instance, in the sensitivity table |
| A9 | Always-on services, per instance | keycloak 1 vCPU / 1.5 GiB; temporal-server 1 / 2; openobserve 1 / 2; prometheus 0.5 / 1; otel-collector 0.25 / 0.5; temporal-ui, pgbouncer and the five workers 0.25 / 0.25 each. **Total 5.5 vCPU, 8.75 GiB** | No requests in `compose.yaml`; small sizes for low-traffic daemons |
| A10 | Records created per seat per month | 400 (scales with A1: 200 / 400 / 800) | About 18 per working day |
| A11 | Stored size of one record | 5 KiB (row, indexes, audit entries, history) | The 800k seed carries about 1 KiB of `data` per record; rounded up for indexes and the audit log |
| A12 | Retention | 12 months of records | Steady state after one year |
| A13 | Base database disk per instance | 20 GiB | WAL (`max_wal_size = 8GB` on M1's host), Keycloak, Temporal, OS |
| A14 | Observability storage | 50 GiB, fixed | Log and metric retention for openobserve and prometheus |
| A15 | Backup copies in object storage | 2 × database data | Full plus incrementals |
| A16 | Egress per request | 20 KiB | About one list page of JSON |

### Unit prices (P rows): **example, not a vendor choice**

One example price set, in the range of on-demand public-cloud list prices. Choosing a vendor, and so the real prices, is the human's call. Replace this table and the `P` constants, then re-run the script.

| ID | Unit | Example price |
|---|---|---|
| P1 | vCPU-month | $20.00 |
| P2 | GiB RAM-month | $2.50 |
| P3 | Block storage, GiB-month | $0.10 |
| P4 | Object storage, GiB-month | $0.023 |
| P5 | Egress, GiB | $0.09 |
| P6 | Load balancer, month | $18.00 |

## Formulas

For seats N and load factor f (0.5, 1, 2):

```
requests per seat-hour   r = A1 × f
peak req/s                 = N × r × A3 / 3600
planning capacity          = M1 200 / 16 vCPU / A4 × A5   = 3.125 mixed req/s per vCPU
load-tier vCPU           L = max(A7, ceil(peak req/s / planning capacity))
total vCPU                 = A8 × (A9 vCPU 5.5 + L)
total GiB RAM              = A8 × (A9 GiB 8.75 + A6 × L)
DB data GiB              d = N × A10 × f × A12 × A11 / 1024²
block GiB                  = A8 × (A13 + d) + A14
object GiB                 = A15 × d
egress GiB                 = N × r × A2 × A16 / 1024²
$/month      = vCPU × P1 + GiB × P2 + block × P3 + object × P4 + egress × P5 + P6
$/seat/month = $/month / N
```

Worked row, 500 seats, expected load:

- peak = 500 × 120 × 3 / 3600 = 50 req/s; L = ceil(50 / 3.125) = 16 vCPU.
- vCPU = 2 × (5.5 + 16) = 43 → $860.00. RAM = 2 × (8.75 + 32) = 81.5 GiB → $203.75.
- d = 500 × 400 × 12 × 5 / 1024² = 11.4 GiB. Block = 2 × (20 + 11.4) + 50 = 112.9 GiB → $11.29. Object 22.9 GiB → $0.53.
- Egress = 500 × 120 × 176 × 20 / 1024² = 201.4 GiB → $18.13. Load balancer $18.00.
- Total $1,111.69 / 500 = **$2.22** per seat per month.

## Sizing

| Seats | Load | Peak req/s | Load-tier vCPU | Total vCPU | Total GiB RAM | DB data GiB | Block GiB | Object GiB | Egress GiB |
|---|---|---|---|---|---|---|---|---|---|
| 50 | low | 2.5 | 4 | 19.0 | 33.5 | 0.6 | 91.1 | 1.1 | 10.1 |
| 50 | expected | 5.0 | 4 | 19.0 | 33.5 | 1.1 | 92.3 | 2.3 | 20.1 |
| 50 | high | 10.0 | 4 | 19.0 | 33.5 | 2.3 | 94.6 | 4.6 | 40.3 |
| 500 | low | 25.0 | 8 | 27.0 | 49.5 | 5.7 | 101.4 | 11.4 | 100.7 |
| 500 | expected | 50.0 | 16 | 43.0 | 81.5 | 11.4 | 112.9 | 22.9 | 201.4 |
| 500 | high | 100.0 | 32 | 75.0 | 145.5 | 22.9 | 135.8 | 45.8 | 402.8 |

## Cost per month, by line (example prices)

| Seats | Load | Compute (vCPU) | Memory | Block storage | Backup (object) | Egress | Load balancer | Total $/month | **$/seat/month** |
|---|---|---|---|---|---|---|---|---|---|
| 50 | low | 380.00 | 83.75 | 9.11 | 0.03 | 0.91 | 18.00 | 491.80 | **9.84** |
| 50 | expected | 380.00 | 83.75 | 9.23 | 0.05 | 1.81 | 18.00 | 492.84 | **9.86** |
| 50 | high | 380.00 | 83.75 | 9.46 | 0.11 | 3.63 | 18.00 | 494.94 | **9.90** |
| 500 | low | 540.00 | 123.75 | 10.14 | 0.26 | 9.06 | 18.00 | 701.22 | **1.40** |
| 500 | expected | 860.00 | 203.75 | 11.29 | 0.53 | 18.13 | 18.00 | 1111.69 | **2.22** |
| 500 | high | 1500.00 | 363.75 | 13.58 | 1.05 | 36.25 | 18.00 | 1932.64 | **3.87** |

## Sensitivity: $/seat/month at expected load, one input changed

| Change one input | 50 seats | 500 seats |
|---|---|---|
| none (baseline) | 9.86 | 2.22 |
| A8 single instance (HA = 1) | 5.18 | 1.15 |
| A4 mixed request = 4 creates | 9.86 | 3.82 |
| A4 mixed request = 1 create | 9.86 | 1.42 |
| P1 vCPU price × 2 | 17.46 | 3.94 |
| All P prices × 2 | 19.71 | 4.45 |

## Not in the model

Each would raise the number, and each needs a decision before it can be priced.

- People: operations, on-call, support.
- Managed-service premiums (for example managed PostgreSQL instead of a VM), and any vendor discount or commitment.
- Non-production environments (staging, CI), a disaster-recovery region, taxes.
- Licences: none today; every image in `compose.yaml` is open source or ours.
- A multi-host layout for the 32-vCPU load tier at 500 seats and high load (D4 revisit trigger).

## Status (ADR-008 D9)

- **The method is in the repo**: this page and its script. M-X4 is no longer "planned".
- **The inputs are mostly assumptions** (A1–A16, P1–P6). Only M1–M4 are measured, and M1 was measured on one laptop host without PgBouncer or HTTP. The figures are a model, not evidence of a production cost.
- **X4 stays "Not claimed"** until Q1 sets a target. This page compares no figure with a list price (D4).
- **Triggers to re-run:** the human answers Q1 or picks a vendor (replace P); the first pilot reports requests per seat (replace A1, A3, A4); a production-storage benchmark (`AR-S10-02`) or a throughput test through PgBouncer and HTTP (replace M1); resource requests are added to the deployment (replace A7, A9).
