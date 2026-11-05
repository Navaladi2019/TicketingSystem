package diffreaders

// A dot import of internal/db/audited: LogTable is bare here, and only here
// (TICK-551 item 5).
import . "github.com/Navaladi2019/TicketingSystem/internal/db/audited"

const (
	dotSQL = "SELECT diff_salt FROM " + LogTable // want: diff_salt
	// Look-alike: a safe column through the dot import.
	dotSafeSQL = "SELECT id FROM " + LogTable
)
