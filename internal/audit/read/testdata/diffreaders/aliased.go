package diffreaders

// A named import alias of internal/db/audited (TICK-551 item 6).
import aud "github.com/Navaladi2019/TicketingSystem/internal/db/audited"

const (
	namedAliasSQL = "SELECT diff FROM " + aud.LogTable               // want: diff
	namedRowSQL   = "SELECT to_jsonb(a) FROM " + aud.LogTable + " a" // want: row
	// Look-alike: a safe column through the alias.
	namedSafeSQL = "SELECT diff_sha256 FROM " + aud.LogTable
)
