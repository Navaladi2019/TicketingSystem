// Package s13kit checks the S13 acceptance kit in docs/acceptance/S13
// (TICK-426, P5-04): the sample audit exports a human runs
// `nava audit verify` on, and the output and exit code each must give.
//
// Its tests do three things:
//
//   - TestSamplesAreCurrent rebuilds every sample file from the P5-02 golden
//     vector generator (internal/audit/chain/vectors, through
//     internal/audit/verify/verifytest) and fails if a committed file differs.
//     No hash in a sample is written by hand. To rewrite the samples:
//
//     go test ./tools/acceptance/s13kit -run TestSamplesAreCurrent -update
//
//   - TestSteps builds cmd/nava and runs it on every sample, comparing stdout
//     and the exit code with docs/acceptance/S13/expected/<sample>.txt
//     (first line "exit N", then the exact stdout). Those files are written
//     by hand from the S13 story and are never regenerated.
//
//   - TestReadme fails if docs/acceptance/S13/README.md does not show, for
//     every sample, its command, its exit code and its expected first line.
//
// The package has no non-test code beyond this comment.
package s13kit
