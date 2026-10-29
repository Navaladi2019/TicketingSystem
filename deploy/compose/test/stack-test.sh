#!/bin/sh
# Stack tests. Usage: stack-test.sh <case>   one case, cases/<case>.sh
#                    stack-test.sh           every case, in registry order
# A new test is one new file under cases/ (its "# stack-case:" header says where it
# runs); this script, the Makefile and ci.yml need no edit. See run-cases.sh.
# Needs the stack up (task stack:up); `internal` brings up its own overlay.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
case $# in
0) exec sh "$dir/run-cases.sh" run all ;;
1) [ -f "$dir/cases/$1.sh" ] || { printf 'FAIL: no case %s; cases: %s\n' "$1" "$(sh "$dir/run-cases.sh" list all | tr '\n' ' ')" >&2; exit 1; }
   exec sh "$dir/cases/$1.sh" ;;
*) printf 'usage: stack-test.sh [case]\n' >&2; exit 1 ;;
esac
