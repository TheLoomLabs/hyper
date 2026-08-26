package cli

import "github.com/TheLoomLabs/hyper/internal/exit"

// The exit codes §12 closes at seven members, under the names every call site
// in this package uses: a command answers `ExitRefused`, and what that number
// is is internal/exit's to say.
//
// **The numbers moved one package out the day a second surface had to read the
// same closed set.** §9's MCP server has no exit code to hand a client, so it
// maps each of these onto an envelope — the answer, a protocol error, or a
// Refusal rendered whole — and a mapping that spelled `77` for itself would
// hold a second opinion about what `77` is. The names stay here because a
// command returning one is this package's grammar, and because the alternative
// was five hundred call sites re-spelled to move a constant (§9, internal/exit,
// internal/mcp, issue #196).
const (
	ExitClean       = exit.Clean
	ExitProblems    = exit.Problems
	ExitUsage       = exit.Usage
	ExitStoreLost   = exit.StoreLost
	ExitRefused     = exit.Refused
	ExitInterrupted = exit.Interrupted
	ExitTerminated  = exit.Terminated
)
