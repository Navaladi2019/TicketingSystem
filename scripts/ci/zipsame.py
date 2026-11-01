"""zipsame.py A B: exit 1 when two zips differ in member names or member bytes
(TICK-391). Timestamps, modes and compressed bytes are ignored."""

import sys
import zipfile


def members(path):
    with zipfile.ZipFile(path) as z:
        return {n: z.read(n) for n in z.namelist()}


def main(a, b):
    ma, mb = members(a), members(b)
    diff = sorted(n for n in set(ma) | set(mb) if ma.get(n) != mb.get(n))
    for n in diff:
        state = "only in committed" if n not in mb else "only in rebuilt" if n not in ma else "differs"
        print(f"zip member {state}: {n}")
    return 1 if diff else 0


if __name__ == "__main__":
    sys.exit(main(*sys.argv[1:3]))
