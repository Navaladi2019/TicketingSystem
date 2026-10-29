# Common start of every case script: source it first (`. "$(dirname "$0")/../case-env.sh"`).
# Git Bash would rewrite the container paths given to docker run and compose exec.
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
. "$(dirname "$0")/../lib.sh"
