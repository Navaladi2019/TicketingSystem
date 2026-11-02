Empty default for the read-only fixture mount of `nava-worker-interptest`
(`deploy/compose/compose.test.yaml`). The interpreter stack case in
`internal/workerstack` mounts its own directory through
`NAVA_INTERPTEST_FIXTURES` instead; this one only keeps Docker from creating a
root-owned directory when the variable is unset.
