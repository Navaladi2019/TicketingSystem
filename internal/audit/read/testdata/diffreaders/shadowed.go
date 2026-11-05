package diffreaders

// The local name audited is the decoy package here, not internal/db/audited.
// A merged import scope would let either file's audited.LogTable win for
// both (TICK-551 item 3).
import audited "github.com/Navaladi2019/TicketingSystem/internal/audit/read/testdata/diffreaders/shadow"

// Look-alike: event_log is not audit_log.
const shadowSQL = "SELECT diff, diff_salt FROM " + audited.LogTable
