-- P5-11-E3 (TICK-508): audit_commit_check() is planned on every call
-- (ADR-004-E3 decisions 1 to 4 and 6; PLAN-005-E3 D-E3-2).
--
-- Why the clause exists (E3, cached-plan reuse): plpgsql caches the probe's
-- plan per session. Under plan_cache_mode = auto it switches to a generic
-- plan after five calls, and under force_generic_plan from the first call.
-- When the probe runs while the current month's partition is young (about 2
-- pages), that generic plan is a Seq Scan of the partition. A pooled
-- session keeps it, so a later import of n subjects runs n Seq Scans over a
-- partition that grows to n rows in the same transaction: O(n^2) (measured
-- t(4N)/t(N) 12.3 to 20.0 without the clause, 3.1 to 5.4 with it, N =
-- 1,000). force_custom_plan plans each call with the bound values as
-- constants against the partition's current size, so the probe stays an
-- index probe on subject_id and occurred_at.
--
-- This supersedes the claim in 20261103100000_audit_commit_check.sql's
-- header that the cached generic plan probes the full R2 key. That holds on
-- an analysed partition, not on a young one. That file is not edited.
--
-- The probe's bound values stay plpgsql locals (decision 2: v_tenant, v_at,
-- v_txid, v_type, v_id; no NEW.<col> in the PERFORM), and P5-10b's
-- n_distinct = 1 on occurred_at stays. CI-A3 (internal/db/audited.Check)
-- fails when audit_commit_check's proconfig lacks
-- plan_cache_mode=force_custom_plan, so a later CREATE OR REPLACE FUNCTION
-- without the SET (which resets proconfig) fails the migrator's gate.
--
-- Nothing else changes: no schema, grant, policy, partition, role or
-- database GUC.

-- +goose Up
ALTER FUNCTION audit_commit_check() SET plan_cache_mode = force_custom_plan;

-- +goose Down
ALTER FUNCTION audit_commit_check() RESET plan_cache_mode;
