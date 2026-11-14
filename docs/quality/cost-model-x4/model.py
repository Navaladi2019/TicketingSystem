"""M-X4 cost-per-seat model (ADR-008 X4, TICK-886).

Stdlib only. Run: python -I docs/quality/cost-model-x4/model.py
Prints the tables in docs/quality/cost-model-x4.md. Every constant below is a
row in that document's input tables; change a row here and there together.
"""

import math

# --- Measured inputs (M rows) -------------------------------------------------
M1_CAPACITY_RPS = 200.0      # gaps creates/s carried, p95 33.2 ms, not saturated (TICK-700)
M1_HOST_VCPU = 16.0          # reference host VM, 16 vCPU (acl-perf-800k.md)

# --- Assumptions (A rows) -----------------------------------------------------
A1_REQ_PER_SEAT_HOUR = 120.0  # expected; low x0.5, high x2
LOAD_FACTORS = {"low": 0.5, "expected": 1.0, "high": 2.0}
A2_ACTIVE_HOURS_MONTH = 176.0  # 22 working days x 8 h
A3_PEAK_FACTOR = 3.0           # peak-minute rate / active-hour average
A4_MIX_COST = 2.0              # one mixed request costs this many measured creates
A5_TARGET_UTIL = 0.5           # plan load tier to 50% CPU at peak
A6_GIB_PER_LOAD_VCPU = 2.0     # memory per load-tier vCPU
A7_LOAD_FLOOR_VCPU = 4.0       # nava-core 1 + postgres 3, minimum
A8_HA = 2.0                    # every service runs twice (HA); 1 = single instance
# A9 fixed tier per instance: service -> (vCPU, GiB)
A9_FIXED = {
    "keycloak": (1.0, 1.5),
    "temporal-server": (1.0, 2.0),
    "temporal-ui": (0.25, 0.25),
    "pgbouncer": (0.25, 0.25),
    "nava-worker": (0.25, 0.25),
    "nava-relay": (0.25, 0.25),
    "nava-purge": (0.25, 0.25),
    "nava-onboard": (0.25, 0.25),
    "nava-groupsync": (0.25, 0.25),
    "otel-collector": (0.25, 0.5),
    "prometheus": (0.5, 1.0),
    "openobserve": (1.0, 2.0),
}
A10_RECORDS_PER_SEAT_MONTH = 400.0  # expected; scales with the load factor
A11_KIB_PER_RECORD = 5.0            # row + indexes + audit + history
A12_RETENTION_MONTHS = 12.0         # steady state after one year
A13_BASE_DB_GIB = 20.0              # WAL (max_wal_size 8GB), Keycloak, Temporal, OS
A14_OBS_GIB = 50.0                  # observability storage, fixed
A15_BACKUP_COPIES = 2.0             # object storage = data x this
A16_KIB_PER_RESPONSE = 20.0         # egress per request

# --- Example unit prices (P rows): example, not a vendor choice ------------------
P = {
    "vcpu_month": 20.0,
    "gib_ram_month": 2.50,
    "block_gib_month": 0.10,
    "object_gib_month": 0.023,
    "egress_gib": 0.09,
    "lb_month": 18.0,
}

KIB_PER_GIB = 1024.0 * 1024.0


def model(seats, load, prices=P, ha=A8_HA, mix=A4_MIX_COST):
    f = LOAD_FACTORS[load]
    req_hour = A1_REQ_PER_SEAT_HOUR * f
    peak_rps = seats * req_hour * A3_PEAK_FACTOR / 3600.0
    rps_per_vcpu = M1_CAPACITY_RPS / M1_HOST_VCPU / mix * A5_TARGET_UTIL
    load_vcpu = max(A7_LOAD_FLOOR_VCPU, math.ceil(peak_rps / rps_per_vcpu))
    fixed_vcpu = sum(v for v, _ in A9_FIXED.values())
    fixed_gib = sum(g for _, g in A9_FIXED.values())
    vcpu = ha * (fixed_vcpu + load_vcpu)
    gib = ha * (fixed_gib + A6_GIB_PER_LOAD_VCPU * load_vcpu)
    data_gib = (seats * A10_RECORDS_PER_SEAT_MONTH * f * A12_RETENTION_MONTHS
                * A11_KIB_PER_RECORD / KIB_PER_GIB)
    block_gib = ha * (A13_BASE_DB_GIB + data_gib) + A14_OBS_GIB
    object_gib = A15_BACKUP_COPIES * data_gib
    egress_gib = (seats * req_hour * A2_ACTIVE_HOURS_MONTH
                  * A16_KIB_PER_RESPONSE / KIB_PER_GIB)
    lines = {
        "compute (vCPU)": vcpu * prices["vcpu_month"],
        "memory": gib * prices["gib_ram_month"],
        "block storage": block_gib * prices["block_gib_month"],
        "backup (object)": object_gib * prices["object_gib_month"],
        "egress": egress_gib * prices["egress_gib"],
        "load balancer": prices["lb_month"],
    }
    total = sum(lines.values())
    return {
        "peak_rps": peak_rps, "load_vcpu": load_vcpu, "vcpu": vcpu, "gib": gib,
        "data_gib": data_gib, "block_gib": block_gib, "object_gib": object_gib,
        "egress_gib": egress_gib, "lines": lines, "total": total,
        "per_seat": total / seats,
    }


def main():
    print("## Sizing\n")
    print("| Seats | Load | Peak req/s | Load-tier vCPU | Total vCPU | Total GiB RAM | DB data GiB | Block GiB | Object GiB | Egress GiB |")
    print("|---|---|---|---|---|---|---|---|---|---|")
    for s in (50, 500):
        for l in LOAD_FACTORS:
            r = model(s, l)
            print(f"| {s} | {l} | {r['peak_rps']:.1f} | {r['load_vcpu']:.0f} | {r['vcpu']:.1f} | {r['gib']:.1f} | "
                  f"{r['data_gib']:.1f} | {r['block_gib']:.1f} | {r['object_gib']:.1f} | {r['egress_gib']:.1f} |")
    print("\n## Cost per month (example prices)\n")
    names = list(model(50, "low")["lines"])
    print("| Seats | Load | " + " | ".join(names) + " | Total $/month | **$/seat/month** |")
    print("|---|---|" + "---|" * len(names) + "---|---|")
    for s in (50, 500):
        for l in LOAD_FACTORS:
            r = model(s, l)
            cells = " | ".join(f"{r['lines'][n]:.2f}" for n in names)
            print(f"| {s} | {l} | {cells} | {r['total']:.2f} | **{r['per_seat']:.2f}** |")
    print("\n## Sensitivity, $/seat/month at expected load\n")
    print("| Change one input | 50 seats | 500 seats |")
    print("|---|---|---|")
    base = {s: model(s, "expected")["per_seat"] for s in (50, 500)}
    print(f"| none (baseline) | {base[50]:.2f} | {base[500]:.2f} |")
    rows = [
        ("A8 single instance (HA = 1)", dict(ha=1.0)),
        ("A4 mixed request = 4 creates", dict(mix=4.0)),
        ("A4 mixed request = 1 create", dict(mix=1.0)),
        ("P vCPU price x2", dict(prices={**P, "vcpu_month": P["vcpu_month"] * 2})),
        ("P all prices x2", dict(prices={k: v * 2 for k, v in P.items()})),
    ]
    for name, kw in rows:
        print(f"| {name} | {model(50, 'expected', **kw)['per_seat']:.2f} | {model(500, 'expected', **kw)['per_seat']:.2f} |")
    fixed_v = sum(v for v, _ in A9_FIXED.values())
    fixed_g = sum(g for _, g in A9_FIXED.values())
    print(f"\nFixed tier per instance: {fixed_v} vCPU, {fixed_g} GiB.")
    print(f"Planning capacity: {M1_CAPACITY_RPS / M1_HOST_VCPU / A4_MIX_COST * A5_TARGET_UTIL:.4f} mixed req/s per vCPU.")


if __name__ == "__main__":
    main()
