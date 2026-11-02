#!/bin/sh
# S13 acceptance kit (TICK-426): run bin/nava audit verify on every sample in
# docs/acceptance/S13 and compare stdout and the exit code with
# docs/acceptance/S13/expected/<sample>.txt (first line "exit N", then the
# exact stdout), and require an empty stderr, as TestSteps does. Run from
# the repository root after `task build`.
# Exits 1 if any sample differs.
set -u

kit=docs/acceptance/S13
nava=../../../bin/nava
[ -x bin/nava.exe ] && nava=../../../bin/nava.exe
if [ ! -x "bin/${nava##*/}" ]; then
	echo "s13: bin/nava not found; run task build first" >&2
	exit 2
fi

cd "$kit" || exit 2
errf=$(mktemp) || exit 2
trap 'rm -f "$errf"' EXIT
fail=0
n=0
for f in *.jsonl; do
	name=${f%.jsonl}
	exp=expected/$name.txt
	if [ ! -f "$exp" ]; then
		echo "FAIL $f: no $exp"
		fail=1
		continue
	fi
	want_exit=$(head -n 1 "$exp" | sed 's/^exit //')
	want_out=$(tail -n +2 "$exp")
	got_out=$("$nava" audit verify "$f" 2>"$errf")
	got_exit=$?
	got_err=$(cat "$errf")
	n=$((n + 1))
	if [ "$got_exit" = "$want_exit" ] && [ "$got_out" = "$want_out" ] && [ -z "$got_err" ]; then
		echo "PASS $f (exit $got_exit)"
	else
		echo "FAIL $f"
		echo "  want exit $want_exit:"
		echo "$want_out" | sed 's/^/    /'
		echo "  got exit $got_exit:"
		echo "$got_out" | sed 's/^/    /'
		if [ -n "$got_err" ]; then
			echo "  stderr (want empty):"
			echo "$got_err" | sed 's/^/    /'
		fi
		fail=1
	fi
done

if [ "$fail" -ne 0 ]; then
	echo "s13: FAIL"
	exit 1
fi
echo "s13: all $n samples match"
