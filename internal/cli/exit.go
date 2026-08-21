package cli

// The exit codes §12 closes at seven members, one per way an invocation can
// end, each carrying the outcome it maps onto. They are declared here in full
// rather than as each milestone reaches one, so that the milestone that first
// reaches a code inherits the name the closed set already fixed instead of
// minting a second constant for the same number (issue #102).
//
// 75 and 77 are sysexits' EX_TEMPFAIL and EX_NOPERM, and the pairing carries
// the difference: 75 says retry me, 77 says a verbatim retry will refuse
// identically. What sorts a stop into one or the other is whether an act is
// required to clear it — an edit, an init, a project, a newer binary — and
// never how severe it was (ADR-0061).
const (
	// ExitClean — the command did what it was asked, including a Run whose
	// every Step skipped.
	ExitClean = 0

	// ExitProblems — a Run the world resisted, or a command that is not a
	// Run reporting problems it found. `check` lands here with one problem
	// row or a thousand.
	ExitProblems = 1

	// ExitUsage — a usage error. No Run began, no row stream opens, and no
	// member of the outcome triple applies: an unknown flag, an unresolvable
	// repository root, a positional matching nothing.
	ExitUsage = 2

	// ExitStoreLost — a Run that lost the Store: to the lock, to the sync at
	// Run start, or to a push it could not rebase through in three attempts.
	// All three are reachable (issue #138); none of them is a Refusal, and
	// none is the world resisting the work.
	ExitStoreLost = 75

	// ExitRefused — a guardrail declined before any effect reached the
	// world. The version pin gate and an absent Store carry it from any
	// command that hits them; a Run that refuses carries it too.
	ExitRefused = 77

	// ExitInterrupted — a Run stopped by an interrupt, having drained
	// (ADR-0015). Unreachable until the Run exists (milestone 5).
	ExitInterrupted = 130

	// ExitTerminated — a Run stopped by a termination signal, drained the
	// same way. Unreachable until the Run exists (milestone 5).
	ExitTerminated = 143
)
