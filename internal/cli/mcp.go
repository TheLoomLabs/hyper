package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/mcp"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// RunMCP implements `hyper mcp` — the invocation that starts §9's second
// surface, and the **third command outside §9's tree of sixteen**, beside
// `version` and `completions` (§9, ADR-0088, issue #195).
//
// **It takes no arguments at all.** No `--repo-dir`, no `--json`, no transport
// flag, no port: an argument here would be a per-server setting, and the layer
// that has none is the point. Not even the three globals are offered — a
// `--repo-dir` would name a repository nothing at this layer reads, and there
// is no answer here for `--json` or `--no-color` to present — so anything on
// the line is a usage error decided from the argument list alone, before a
// server exists (§9, ADR-0014, ADR-0088).
//
// **It resolves no repository, and it is not a fourth exemption from the pin
// gate.** The invocation is not the act: what acts on a repository is each tool
// the server carries, and every one of them resolves one and passes the gate
// exactly as the command it carries does, at the moment it resolves one
// (ADR-0088). What the process fixes is *which* repository — `HYPER_REPO_DIR`
// where the environment sets it, otherwise the walk up from the working
// directory bounded by the git root, both of them the **process's**, fixed by
// the client that started it. A client that wants a second repository starts a
// second server, which is what one process per client already means.
//
// **The stdout it is not handed is the whole of its stream discipline.** §9's
// *stdout is the answer* is not true of this process — the frames are — so the
// signature takes the narration and nothing else: there is no writer here to
// hand a command, and the destination behind every tool has none either
// (destination.go, mcp.Serve). A usage error still writes a human sentence,
// and it writes it where every human sentence goes.
func RunMCP(args []string, narrate io.Writer, process Process, facts version.Facts) int {
	if len(args) > 0 {
		fmt.Fprintf(narrate, "hyper mcp: takes no arguments, got %s\n", args[0])
		return ExitUsage
	}

	// The context is the process's own and carries no deadline: the server
	// dies with the client, which is the transport reaching end of input,
	// and a bound invented here would be a session length nobody agreed to
	// (§9, §13).
	if err := MCPServer(process, facts).Serve(context.Background()); err != nil {
		fmt.Fprintf(narrate, "hyper mcp: %s\n", err)
		return ExitProblems
	}
	return ExitClean
}

// MCPServer is the MCP server over this process: the tool set internal/mcp
// states, at the binary's own version, over the dispatch below.
//
// It is exported for Streams' own reason — a caller outside the package that
// drives the surface rather than the command line needs a door, and this is
// it — and the golden corpus is that caller: a `call` case is driven through
// this server over in-memory transports, so what the corpus exercises and what
// a client starts are one server assembled one way (golden_mcp_test.go).
//
// It is not spelled `Server`, and the reason is the word rather than the
// package: §9 and §13 each spend a paragraph refusing `serve`, and ADR-0088
// refuses it again for the invocation. A bare `Server` here would put the word
// back at the one place a reader looks up what the binary does — which is the
// refusal working as stated rather than an inconvenience of it.
func MCPServer(process Process, facts version.Facts) *mcp.Server {
	return mcp.NewServer(facts.Version, MCPDispatch(process, facts))
}

// MCPDispatch is what stands behind every tool: one call's argv run through the
// same table `hyper` dispatches a command line through, against a destination
// that holds no stream (destination.go).
//
// It is exported for MCPServer's own reason, one layer down. The drivers that
// reach past a golden drive a corpus case with one more input supplied, and the
// input a **cancelled call** supplies is the call's own context — which is the
// server's to make and not a caller's, so a driver that wanted to hold it had
// nowhere to stand. A driver stands the same tool set over this dispatch with
// an observer in front of it, exactly as the corpus stands the same server with
// a tee on the wire (mcp.Server.Call, mcp_cancelled_test.go, issue #202).
func MCPDispatch(process Process, facts version.Facts) mcp.Dispatch {
	// **The server installs no signal watch**, and this is where that
	// becomes true rather than merely intended: the process's signals belong
	// to the client that spawned it and not to any one call in flight, so a
	// Run here is one nobody interrupts by signal and exit codes `130` and
	// `143` are unreachable from this surface (§6, §9, §12, ADR-0015,
	// ADR-0092).
	//
	// It costs nothing, because the stop this surface does have is the
	// cancelled call below: the envelope carries `failed` and the Journal
	// carries the truthful account, which is what the codes would have added
	// a number to.
	//
	// It is cleared **once**, here rather than inside the dispatch: the
	// value behind a server is one, calls arrive on goroutines of their own,
	// and a member written per call would be a write two of them race on.
	process.Notify = nil

	return func(call mcp.Call) mcp.Answer {
		// The destination that holds no stream, made fresh per call: what
		// it retains is one command's answer, and a server handing two
		// calls one buffer would be a surface whose second answer carried
		// the first's page.
		//
		// It carries the call's two protocol facts because it is the
		// **surface's** value: where a Step boundary goes is where this
		// answer's narration goes, and a caller that has stopped waiting
		// for an answer is a destination that has gone away (§9,
		// destination.go).
		to := &collected{progress: call.Progress, cancelled: call.Context.Err}
		code, dispatched := runRepositoryCommand(call.Argv, to, process, facts)
		if !dispatched {
			// Unreachable by construction, and stated rather than
			// assumed: every tool is named for one of §9's sixteen and
			// builds that command's own line, so a name arriving here
			// that the table does not hold is this repository's bug and
			// not a caller's. It is written where a usage error is
			// written and carried the way one is — the message reaches
			// a caller as the protocol error §9 answers a malformed
			// call with, which is the nearest true thing to say about a
			// tool that named nothing (issue #196).
			fmt.Fprintf(to.narrate(), "hyper mcp: %q is not one of §9's sixteen commands; the tool set names a command this binary does not dispatch\n", strings.Join(call.Argv, " "))
			code = ExitUsage
		}
		// One construction of the Answer whichever way the call went, so
		// that every buffer the destination kept crosses the boundary and
		// the mapping on the other side decides which of them the envelope
		// is composed from (mcp.Answer, envelopeOf).
		return mcp.Answer{
			Rows:      to.rows,
			Terminal:  to.terminal,
			Rendering: to.rendering.String(),
			Refusal:   to.refusalRendering.String(),
			Narration: to.narration.String(),
			Exit:      code,
		}
	}
}
