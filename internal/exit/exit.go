// Package exit is §12's closed set of exit codes: seven members, one per way an
// invocation can end, each carrying the outcome of the triple it maps onto (§9,
// §12).
//
// **It is a package because two surfaces read the same set.** The CLI returns
// one of these from every invocation, and §9's MCP server maps each one onto a
// return envelope — *the answer, a protocol error, or a Refusal rendered whole*
// — on a surface that has no exit code of its own to hand back (§9, issue
// #196). A mapping carrying its own spelling of `77` would be a second opinion
// about what `77` is, held one package away from the commands that return it.
//
// The numbers are declared in full rather than as each milestone reaches one,
// so that the milestone which first reaches a code inherits the name the closed
// set already fixed instead of minting a second constant for the same number
// (issue #102).
//
// 75 and 77 are sysexits' EX_TEMPFAIL and EX_NOPERM, and the pairing carries
// the difference: 75 says retry me, 77 says a verbatim retry will refuse
// identically. What sorts a stop into one or the other is whether an act is
// required to clear it — an edit, an init, a project, a newer binary — and
// never how severe it was (ADR-0061).
package exit

const (
	// Clean — the command did what it was asked, including a Run whose
	// every Step skipped.
	Clean = 0

	// Problems — a Run the world resisted, or a command that is not a Run
	// reporting problems it found. `check` lands here with one problem row
	// or a thousand.
	Problems = 1

	// Usage — a usage error. No Run began, no row stream opens, and no
	// member of the outcome triple applies: an unknown flag, an unresolvable
	// repository root, a positional matching nothing.
	Usage = 2

	// StoreLost — a Run that lost the Store: to the lock, to the sync at Run
	// start, or to a push it could not rebase through in three attempts. All
	// three are reachable (issue #138); none of them is a Refusal, and none
	// is the world resisting the work.
	StoreLost = 75

	// Refused — a guardrail declined before any effect reached the world.
	// The version pin gate and an absent Store carry it from any command
	// that hits them; a Run that refuses carries it too.
	Refused = 77

	// Interrupted — a Run stopped by an interrupt, having drained: the Step
	// in flight finished, no further Step started, and the Run closed its
	// own entry `failed` (§6, ADR-0015, issue #145).
	Interrupted = 130

	// Terminated — a Run stopped by a termination signal, drained the same
	// way and `failed` the same way. The two codes differ by which signal
	// arrived and by nothing else (§12).
	Terminated = 143
)
