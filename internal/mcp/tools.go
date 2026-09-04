package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The tool set (§9, issues #195, #197, #198, #199, #200, #201 and #203).
//
// **A tool is a schema, an argv, and nothing else.** Each declares its
// arguments typed and closed exactly as the flag or the positional it carries
// is, builds the command line its command would have received, and hands it to
// the same dispatch behind the server's destination. §9 fixes that *ergonomics
// is the whole of the difference between the two*; a tool that reached past the
// argv would be a second place for a guardrail to be skipped, a Refusal to be
// reworded or a row to be reshaped.
//
// §9 states thirteen tools, each named for the command it carries, and all
// thirteen are here. The table is what made that true one milestone at a time,
// on tree.go's own rule for the command surface — a name is real when the code
// behind it is — and it stays the one place a fourteenth would have to be
// written rather than a list to be kept in step.

// tool is one entry of the set: what it is called, what a client is told it is
// for, the two schemas, and the command line it builds.
//
// The schemas are hand-written JSON rather than inferred from a Go type. That
// is the cost of the low-level registration and it is a cost worth paying twice
// over: §9's arguments are closed sets and enums that inference would widen to
// bare strings, and an `outputSchema` declared here is declared **once and for
// every call of the tool** rather than composed per answer.
type tool struct {
	name        string
	description string
	input       json.RawMessage
	output      json.RawMessage
	// argv reads the raw arguments and builds the command line. It answers
	// an error where the arguments do not satisfy the schema, and that error
	// becomes a JSON-RPC error: an argument violating a schema is a
	// malformed call (§9, server.go).
	argv func(arguments json.RawMessage) ([]string, error)
	// text is §9's text-block table read as a property of the tool, which is
	// how §9 states four of its five rows: *any ordinary return* carries one
	// summary line, **`check`** carries its rows beneath that line,
	// **`runs`** and **`records`** carry where the record lives beneath it,
	// and **`review`** carries the full rendered review surface. The
	// remaining row is a Refusal's, which is a property of the path rather
	// than of the tool and is therefore not here.
	//
	// It is a member on the tool rather than a composition the tool
	// supplies, because what it selects between is three readings of one
	// answer the destination already retained — the rows counted, the rows
	// counted and then drawn, or the page the command wrote
	// (destination.go). A tool that composed its own text would be a tool
	// holding a rendering, which is the one thing *a tool is a schema, an
	// argv, and nothing else* forbids.
	//
	// It is one member rather than two bits for the reason the table has
	// cases rather than axes: *the whole page* and *the rows beneath the
	// line* are alternatives, and a pair of booleans has a fourth state that
	// names no row of §9's table at all.
	text textBlock
	// executes is §9's execution half read as a property of the tool: `run`
	// is the one of the thirteen whose answer carries §12's triple, and a
	// tool that leaves this nil is one that carries no `outcome` key at all
	// (§9, envelope.go, issue #200).
	//
	// It is a **function of the arguments** rather than a bare bit, because
	// what the envelope needs from the tool is a pair rather than a fact:
	// the outcome is §12's reading of the exit code, and `dry_run` is this
	// call's own — written wherever `outcome` is, and never guessed. It is
	// read only on the one path where the command wrote no terminal row to
	// carry either (executionOf).
	//
	// It reads the arguments through the same reader argv reads them
	// through, which is what keeps the two from disagreeing about one call:
	// there is one reading of a `run`'s arguments and it is asked twice
	// (runArguments).
	executes func(arguments json.RawMessage) (execution, error)
}

// textBlock is which shape an ordinary return's `text` takes on this tool:
// §9's asymmetric table, minus the row a Refusal takes on every tool at once.
//
// The zero value is the table's first row, which is what makes a tool that
// declares nothing here the ordinary case — eleven of the thirteen — rather
// than a tool that forgot to.
type textBlock int

const (
	// summaryLine is *one summary line, outcome first*: the counted rows,
	// or §8's terminal line as a sentence where the tool is a Run. It is
	// what a listing carries, because a listing is a result set and not a
	// remediation.
	summaryLine textBlock = iota
	// rowsBeneathSummary is the summary line and, under it, the rows as the
	// command's own renderer drew them.
	//
	// **It is `check` and nothing else** (issue #214). What earns a tool
	// this row is that its rows are the *remediation path* rather than an
	// answer: a `problem` row is a file, a line, a column, a field, an
	// `error_code` and a message, and an agent that is told how many of
	// them there are and not what they say cannot repair what it wrote — it
	// can only guess, which is what a headless session did across 180 tool
	// calls. That is the same argument §9 already makes for a Refusal, over
	// the return an agent meets far more often.
	//
	// It is not *every ordinary return carrying rows*, which would put a
	// full table in the text block of every listing and say twice what the
	// structured half says once, and it is not *every return with
	// `isError`*, which would append a Run's page to the summary line §9
	// composes for `run` on purpose.
	rowsBeneathSummary
	// locationBeneathSummary is the summary line and, under it, where the
	// record the tool just read is held — store.Location, the sentence the
	// two commands' own pages begin with (ADR-0113, issue #233).
	//
	// **It is `runs` and `records`, the two tools whose job is finding
	// something in the Store.** The CLI half writes that sentence on the
	// page, and the page is what this block is the analogue of: a client is
	// not obliged to surface `structuredContent` and most do not, so a fact
	// carried only there is a fact an agent may never meet. That is
	// `check`'s argument at a second return, and it is the one the record's
	// own location needs most — the fact has no row to be read off, being a
	// constant of the design rather than a result.
	//
	// It is a case of its own rather than a clause of the summary line
	// because the summary line is *composed from the answer*: it counts the
	// rows by their own discriminator, and a tool that appended a sentence
	// to it would be a tool holding a rendering. This is §9's table gaining
	// a row, which is where a text block's shape is decided.
	locationBeneathSummary
	// wholeRendering is the command's page, byte for byte, in place of any
	// line of this surface's own. It is `review`'s, and the promise is what
	// makes it a case of its own: what the block carries is what the
	// command wrote to stdout, so a human reviewer can be handed it
	// verbatim (§9, ADR-0026).
	//
	// **It is the one case that also writes into `structuredContent`**, and
	// a tool that declares it declares a `rendering` member to carry the
	// page there too (§9, ADR-0100, issue #217). The other two cases compose
	// their block out of members that half already carries and this one
	// composes it out of nothing there at all; why that means the block is
	// not enough on its own is Structured.Rendering's to say.
	wholeRendering
)

// declaration is the tool as the SDK publishes it in `tools/list`. It is the
// one place a tool of this package's is expressed in the SDK's terms, and what
// crosses is two strings and two schemas — no handler, no domain type, and
// nothing for the SDK to reflect over.
func (t tool) declaration() *sdk.Tool {
	return &sdk.Tool{
		Name:         t.name,
		Description:  t.description,
		InputSchema:  t.input,
		OutputSchema: t.output,
	}
}

// tools is the set, in §9's own order: Discovery first, and within it the order
// §9's table states, then the repository, then Authoring, then Execution, then
// Inspection, and Lifecycle's `project` last (issue #203).
var tools = []tool{providersTool, providerTool, operationTool, targetsTool, checkTool, reviewTool, runTool, probeTool, runsTool, runShowTool, changesTool, recordsTool, projectTool}

// providersTool carries `hyper providers` — §9's first discovery question,
// *which Provider*, and the one an agent asks before it can write a
// `provider:` at all.
//
// **It takes no arguments**, and neither does the command line it builds. §9
// gives it none: `providers` enumerates a namespace and resolves no name in
// one, and the `--limit` its command carries is not offered here — an argument
// naming a cap would be a per-call setting on a surface whose whole account of
// a result too large is the truncation marker (§9). The command's own default
// applies, and a listing it cuts says so in both halves of the envelope.
var providersTool = tool{
	name:        "providers",
	description: "List every Provider this repository can load, built-in and Extension alike.",
	input:       noArguments,
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "name", "origin", "summary", "operation_count", "digest"],
				"properties": {
					"type": {"const": "provider"},
					"name": {"type": "string"},
					"origin": {"enum": ["built-in", "extension"]},
					"summary": {"type": "string"},
					"operation_count": {"type": "integer", "minimum": 0},
					"digest": {"type": "string"}
				}
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		if err := readArguments(arguments, &struct{}{}); err != nil {
			return nil, err
		}
		return []string{"providers"}, nil
	},
}

// providerTool carries `hyper provider <name>` — §9's second discovery
// question, *which Operation*, and the first tool that takes a name and
// resolves it.
//
// Its one argument is the command's one positional, typed as the positional is:
// a Provider name, matched byte-exact over UTF-8 against the repository's
// Provider namespace. It carries no `--limit` because the command carries none
// — a Manifest is named rather than ranged over — so `truncated` is the bare
// `false` on every call.
var providerTool = tool{
	name:        "provider",
	description: "Report one Provider's Manifest facts and every Operation it exposes.",
	input: closedObject(`{
		"name": {"type": "string", "minLength": 1, "description": "The Provider's name, as its Manifest declares it."}
	}`, "name"),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"oneOf": [
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "digest"],
						"properties": {
							"type": {"const": "manifest"},
							"auth_scheme": {"type": "string"},
							"capabilities_required": {"type": "array", "items": {"type": "string"}},
							"digest": {"type": "string"},
							"schema_version": {"type": "integer"},
							"origin_ref": {"type": "string"},
							"origin_digest": {"type": "string"}
						}
					},
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "name", "kind", "opaque", "summary"],
						"properties": {
							"type": {"const": "operation"},
							"name": {"type": "string"},
							"kind": {
								"enum": ["read", "mutate", "destroy", ""],
								"description": "The Operation's declared Kind. The empty string is a Manifest that declared none, which is a fault the check surface reports rather than a fourth Kind: the row states what the artefact stated."
							},
							"opaque": {"type": "boolean"},
							"summary": {"type": "string"}
						}
					}
				]
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Name string `json:"name"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		if err := namesSomething("name", named.Name, "Provider"); err != nil {
			return nil, err
		}
		// The name goes past a `--`, which is what a command line carrying
		// an arbitrary positional looks like: a Provider called `--json`
		// is a name the namespace can hold, and a tool that handed it to
		// the parser bare would let an argument turn into a flag. The
		// parser already states the grammar — "--" ends the flags — so
		// this is the command line the command would have received rather
		// than a guard invented here (flags.go).
		return []string{"provider", "--", named.Name}, nil
	},
}

// operationTool carries `hyper operation <provider> <operation>` — §9's third
// discovery question, *how do I call it*, and the moment on this surface where
// bytes matter rather than facts: an agent about to write a Definition is
// handed the Manifest's own declaring lines to author against.
//
// **Its two arguments are the command's two positionals, and neither is
// optional.** §9 gives Discovery three tools rather than one taking optional
// arguments for the reason the three commands are three — they are three
// questions asked in order — and here the protocol says it too: *an
// `outputSchema` is declared once and for every call of the tool*, which a tool
// answering a Manifest's rows or an Operation's detail by which argument was
// omitted could not do. It carries no `--limit` because the command carries
// none: one Operation is named, so there is no result set for a cap to cut, and
// `truncated` is the bare `false` on every call.
//
// The two names resolve in order and against two different namespaces, which is
// why either of them matching nothing is a **protocol error** and not one
// answer with a bit set: both are well-typed and name nothing, which is §9's
// third member of the malformed set (§9, ADR-0060, envelope.go, issue #197).
var operationTool = tool{
	name:        "operation",
	description: "Report one Operation's declaring Manifest lines, verbatim, and the facts hyper derives from them.",
	input: closedObject(`{
		"provider": {"type": "string", "minLength": 1, "description": "The Provider's name, as its Manifest declares it."},
		"operation": {"type": "string", "minLength": 1, "description": "The Operation's name, as a key of that Manifest's own operations: block."}
	}`, "provider", "operation"),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "source", "derived"],
				"properties": {
					"type": {"const": "operation_detail"},
					"source": {
						"type": "string",
						"description": "The Manifest lines declaring this Operation, verbatim: the format a caller is expected to author Definitions in, returned rather than re-rendered."
					},
					"derived": {
						"type": "object",
						"additionalProperties": false,
						"required": ["patterns_resolved", "concurrency_limit"],
						"description": "What hyper derives from the declaration above, which the source does not carry in this form. capabilities, bound and repeatability are absent only where a Manifest hyper could not read left nothing to derive them from, which is the check surface's to report.",
						"properties": {
							"capabilities": {
								"type": "array",
								"items": {"type": "string"},
								"description": "The one Capability the Operation's request is written under; an Operation uses exactly one."
							},
							"bound": {
								"enum": ["mandatory", "illegal", "none"],
								"description": "Three members and never a boolean: a destroy Step's Bound is mandatory, an opaque destroy's is illegal — writing one there is refused — and on a read or a mutate none is required. false would carry you need not write one and writing one is refused under one value, on the most severe Operation the tool runs."
							},
							"patterns_resolved": {
								"type": "array",
								"items": {"enum": ["pagination", "polling", "retry"]},
								"description": "The members of the Pattern set this Operation declares, and [] rather than absent where it declares none: a caller asking which Patterns run around this call is answered none of them, which is a fact."
							},
							"record_cardinality": {
								"enum": ["series", "one"],
								"description": "Absent, together with record_identity, on a destroy, which projects no Record of its own."
							},
							"record_identity": {"type": "string"},
							"repeatability": {
								"enum": ["repeatable", "skip-if-recorded", "run-once"],
								"description": "The effective value and not the declared one: run-once where an effectful Manifest omits the key, repeatable where a read does."
							},
							"deadline_seconds": {
								"type": "integer",
								"minimum": 0,
								"description": "The deadline in seconds. The wire fixes the unit so that nothing downstream parses a suffix."
							},
							"concurrency_limit": {
								"type": "integer",
								"minimum": 1,
								"description": "The effective limit, present on every Operation: the declared concurrency:, or 1 where absent, and 1 on every mutate and destroy."
							}
						}
					}
				}
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Provider  string `json:"provider"`
			Operation string `json:"operation"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		// The two are refused separately because the namespaces are two —
		// the repository's, and this Manifest's own — which is the same
		// reason the command writes two messages rather than one
		// (operation.go).
		if err := namesSomething("provider", named.Provider, "Provider"); err != nil {
			return nil, err
		}
		if err := namesSomething("operation", named.Operation, "Operation"); err != nil {
			return nil, err
		}
		// Both names go past one `--`, for providerTool's reason read over
		// two positionals: a name spelled like a flag is a name either
		// namespace can hold, and the parser already states that "--" ends
		// the flags (flags.go).
		return []string{"operation", "--", named.Provider, named.Operation}, nil
	},
}

// targetsTool carries `hyper targets` — §9's question about the repository
// rather than about a Provider, and the only surface anywhere that answers *is
// the credential in place*.
//
// **It takes no arguments**, on providersTool's reading: `targets` enumerates a
// namespace and resolves no name in one, and the `--limit` its command carries
// is not offered here. The command's own default applies, and a listing it cuts
// says so in both halves of the envelope.
//
// **The names travel and the values do not**, which is exactly what an agent
// must write into a Target declaration while never seeing a value — the shape
// §3 fixed when it made a literal in a credential position a load error. Why a
// variable is paired with the slot it fills rather than listed bare, and why
// presence is a member that can be absent, are the command's own reasons and
// are written where the row is (targets.go). §9's MCP sketch spelled the pair
// as two flat members, `credential_env` beside a `credentials_present` — the
// same disagreement its `endpoint` was resolved from, and resolved the same
// way, §12's opening rule sending one fact to two wires under one name (§3,
// §9, ADR-0007, issue #197).
//
// **`presence` is a word out of a closed three and was a boolean**, and this
// wire is the reason that change was worth its own reviewed issue: an agent
// parses this schema, and *the variable is set* and *the variable is set to
// nothing* are the two readings it cannot tell apart by squinting at a
// terminal, a `401` being all either of them produced. The member was renamed
// rather than re-typed so that a client written against the boolean finds no
// `present` at all rather than reading the string "absent" as true (§9, §12,
// ADR-0145, issue #264).
var targetsTool = tool{
	name:        "targets",
	description: "List every Target this repository declares, and the environment variables its credentials resolve from — names only, never a value.",
	input:       noArguments,
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "name"],
				"description": "One Target declaration. The four lists are absent rather than empty where the declaration grants nothing of that kind: a declaration granting no http carries no hosts: at all, and one named local carries no auth: block.",
				"properties": {
					"type": {"const": "target"},
					"name": {"type": "string"},
					"hosts": {
						"type": "array",
						"items": {"type": "string"},
						"description": "The host grant, in the declaration's own order. A grant reduced to its first member is not a grant."
					},
					"accepts_kinds": {"type": "array", "items": {"enum": ["read", "mutate", "destroy"]}},
					"grants_capabilities": {"type": "array", "items": {"type": "string"}},
					"credentials": {
						"type": "array",
						"description": "One member per credential slot the declaration carries, which is wider than what a Run checks: a Run resolves the slots its bindings require, and this has no Procedure in hand to narrow by.",
						"items": {
							"type": "object",
							"additionalProperties": false,
							"required": ["slot"],
							"properties": {
								"slot": {"type": "string"},
								"env": {
									"type": "string",
									"description": "The environment variable the slot resolves from — the name, never the value."
								},
								"presence": {
									"enum": ["absent", "empty", "set"],
									"description": "What the environment did with that variable, computed when the tool runs: absent where it does not hold it, empty where it holds it and it has no characters, set otherwise. It is absent, with env, on a slot naming no variable: there is nothing to ask the environment about, and a word would answer a question nothing asked. These are the three the credential gate decides a Run under, so a slot reading set is a slot no Run will Refuse for want of it."
								}
							}
						}
					}
				}
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		if err := readArguments(arguments, &struct{}{}); err != nil {
			return nil, err
		}
		return []string{"targets"}, nil
	},
}

// problemRow is `check`'s row as a schema, written once because **two tools
// answer it**: `check`, whose whole answer is these rows, and `review`, which
// answers them where the artefact under review is found and will not load — an
// artefact hyper could not read is reported rather than rendered (§9,
// review.go).
//
// It is one fragment rather than one shape spelled twice for the reason §8
// gives the row itself: there is one renderer behind the row and there should
// be one schema in front of it. Two copies would be two copies to keep in step,
// and they had already begun to drift — the second was written without the
// descriptions the first carries, which is a client told less about one tool's
// rows than about another's for no reason it could see.
//
// It is a bare object rather than a closedObject because it is not a schema of
// its own: what it describes is a member of `rows`, and the closure it needs is
// the one written into it here.
const problemRow = `{
	"type": "object",
	"additionalProperties": false,
	"required": ["type", "file", "line", "column", "field", "error_code", "message"],
	"description": "One problem, positioned. Rows are ordered by file path and then by line, which is the order the row stream is written in and the order a caller may not re-sort after printing.",
	"properties": {
		"type": {"const": "problem"},
		"file": {"type": "string", "description": "The artefact's path, relative to the repository root, with forward slashes on every platform."},
		"line": {
			"type": "integer",
			"minimum": 0,
			"description": "The 1-indexed line. Zero is a fault with no line to go to — a whole-file comparison — and is the absence the empty field beside it also reads as."
		},
		"column": {
			"type": "integer",
			"minimum": 0,
			"description": "The 1-indexed column, which rides on the wire only: the page has no column for a fact a consumer filters on."
		},
		"field": {"type": "string", "description": "A path into the artefact, in §8's remediation notation — steps[2].bound, auth.token — and empty where the fault has no position more specific than the file."},
		"error_code": {
			"type": "string",
			"description": "One member of §12's closed error_code set, naming the check that declined. It is not enumerated here: the set is §12's and the checks' own, and a second copy of it on this surface would be a copy to keep in step."
		},
		"message": {"type": "string"}
	}
}`

// cadenceGlossMembers is §10's gloss as schema members, written once because
// **two rows carry it**: `review`'s `artefact` row and `project`'s `workflow`
// row, which is the pairing internal/cli already states — one value on the row
// side, and neither row may carry a different group (cadence_gloss.go).
//
// It is one fragment for problemRow's reason: there is one reading of an
// expression behind the three members and there should be one schema in front
// of them. Wherever a Cadence renders, the gloss renders with it and there is
// no surface exempt (ADR-0063), so a rule that is total is one no consumer may
// hold a second copy of — and two schemas spelling it apart would be that copy
// arriving on the wire.
//
// **The three are the whole of what crosses.** §10's three facts about how the
// executor treats the declaration are derived from `cadence` and `phrase`,
// which are already here, so a consumer derives them exactly as a page does;
// and how the parts are *arranged* is each surface's own, which is why nothing
// here is a composed line (§8, §9, §10).
//
// None of the three is required: all three are absent where the row's subject
// declares no recurrence, and absent together where it declares one the grammar
// does not admit.
//
// **Writing it once moved `review`'s published schema**, which is stated rather
// than passed over: `phrase` had no description where it was spelled inline and
// carries one here, so a client that had already read `review`'s
// `outputSchema` is told more about a member than it was. That is the drift the
// fragment exists to stop rather than a cost of stopping it — one of the two
// copies had already been written with less than the other, which is a client
// told less about one tool's rows than about another's for no reason it could
// see (problemRow).
const cadenceGlossMembers = `
	"cadence": {"type": "string", "description": "The recurrence expression exactly as the artefact wrote it."},
	"phrase": {"type": "string", "description": "The expression stated in words: the times of day, the days and the months it selects. It is a total function of the five fields — every expression the grammar admits gets one — and it is never truncated."},
	"rate": {"type": "number", "description": "Runs per month at the two significant figures §10 rounds to. Zero is a rate: an expression the calendar has no instance of matches nothing."}`

// checkTool carries `hyper check [path...]` — the first of §9's two Authoring
// tools, and the first tool on this surface that answers **pass or fail**
// rather than a fact about the repository.
//
// It is positioned so that the next act is an edit: a row is a file, a line, a
// column, the field, the `error_code` §12 names the check by, and a message,
// which is what makes *report and then edit* practical for an agent for the
// same reason it is for a human (§9, ADR-0001).
//
// **So its rows travel in the text block as well as beside it**, which is the
// `check`'s own row of §9's asymmetric table, and no other tool's (textBlock, issue
// #214). A client that surfaces only `content` to the model behind it is the
// common case rather than the broken one, and an agent told *how many* problems
// there are and not *what* they are has, in place of the edit the row already
// describes, a count to binary-search the schema against. The summary line
// stands first and the rows go beneath it untouched; `structuredContent.rows`
// is unchanged, this being a second channel for one row set and not a move.
//
// **`paths` is the CLI's positional list arriving as one typed argument**, and
// it narrows what is *reported* and never what is *loaded*: every rule §4
// states compares one artefact against another, so a subset of a repository is
// not checkable on its own — only reportable on its own (§9, check.go). The
// argument is optional because the command's positional list is, and the
// absent list is every problem the repository has rather than none.
//
// **The paths are repository-relative, which is where the command resolves
// them**: one root, and the same root on both surfaces (ADR-0089). This tool
// still settles nothing — it builds the command line its command would have
// received and holds no logic of its own — and it does not have to, the command
// having stopped reading the argument against the process's working directory.
// That is what makes the argument nameable here at all: a client picks the
// directory it starts a server in, nothing in the protocol lets a caller see or
// set it, and an argument read against it would mean something different per
// client for one string. What a caller has in hand instead is the path a
// `problem` row above already carries, which is the same spelling going back in
// (§9, check.go, issue #205).
//
// **There is no argument for a `--fix`**, because there is no `hyper check
// --fix`: a checker that can also mutate is a checker you stop trusting, and a
// repair flag on a gate is the shape ADR-0001 removed (§9). It carries no
// `--limit` either, its command carrying none — a repository's problems are
// not a result set to range over — so `truncated` is the bare `false` on every
// call.
var checkTool = tool{
	name:        "check",
	description: "Run every static rule this repository states and report each problem by file, line and error_code.",
	text:        rowsBeneathSummary,
	input: closedObject(`{
		"paths": {
			"type": "array",
			"items": {"type": "string", "minLength": 1},
			"description": "Paths as the command's own positionals take them: repository-relative, or absolute and inside the repository — a path resolving outside it names nothing this repository holds and is refused. Every artefact still loads and only the problems positioned in the ones named are reported; omit it to report every problem the repository has."
		}
	}`),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": `+problemRow+`
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Paths []string `json:"paths"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		// A member is refused for the reason a name is, and here it is
		// the schema saying one layer above what the command says for the
		// same argument: an empty path names no file in the repository,
		// which the command declines in its own sentence and this refuses
		// before a command line is built at all (ADR-0089, issue #205).
		// The index is in the message because the list has more than one
		// place to be wrong in.
		for i, path := range named.Paths {
			if err := namesSomething(fmt.Sprintf("paths[%d]", i), path, "path"); err != nil {
				return nil, err
			}
		}
		if len(named.Paths) == 0 {
			return []string{"check"}, nil
		}
		// The paths go past one `--`, for providerTool's reason read over a
		// list: a path is an arbitrary string, a file called `--json` is one
		// a repository can hold, and the parser already states that "--"
		// ends the flags (flags.go).
		return append([]string{"check", "--"}, named.Paths...), nil
	},
}

// reviewTool carries `hyper review <artefact>` — the second Authoring tool, and
// **the one tool whose text block is the page and nothing else**: its `text` is
// the whole rendered review surface, the gutter and `AUTHORITY` and `FLAGS`
// exactly as the command writes them to stdout. `check` is named by §9's
// text-block table too and is the other shape — a line of this surface's own,
// and then the page (textBlock, issue #214).
//
// That is the point of the tool rather than a convenience of it. An agent can
// read what a human reviewer will read and hand it to them verbatim, before
// asking them to read it — which is the same trade §8 makes for a Refusal, and
// it is made for the same reason: with no bypass anywhere the rendering is the
// whole of what a reviewer is given (§9, ADR-0001, ADR-0026).
//
// **So the page travels on both channels, and the output schema declares the
// member that carries it** — `rendering`, required, above the rows (§9,
// ADR-0100, issue #217). It is the same string the text block carries and not a
// second composition. What makes the second channel necessary is that the rows
// are what the page is drawn *from*, so a caller handed only them is handed the
// ingredients of the thing this tool exists to answer with — a sealed
// acceptance run found exactly that (ADR-0099), and the argument from MCP's own
// asymmetry is at Structured.Rendering.
//
// Three of the command's own rules carry over unchanged, and each is a rule
// about what is *not* an error here. It runs offline against a repository whose
// Store is unreachable, the header naming that absence once rather than the
// tool failing. **A `FLAGS` row is a fact about the artefact rather than a
// problem with it**, so a review that rendered is `isError: false` however many
// flags it carried. And an artefact that loads and **names** one that is not
// there renders, marks `unresolved`, and is not an error — the fault is
// `check`'s to report and this surface's to annotate (§8, ADR-0064).
//
// **An artefact that is not there at all has no row to write**, which is the
// usage error the command spends `2` on and arrives here as a JSON-RPC error:
// a positional that satisfies every schema and still matches nothing is §9's
// third member of the malformed set (§9, ADR-0060, envelopeOf). It is a
// different refusal from the empty string namesSomething catches below, and
// they are told apart by where each is decided — one never reaches a
// repository and the other is what a repository answered.
//
// Its one argument is the command's one positional, typed as the positional is
// and carrying both of its forms: a repository-relative path — one containing
// `/` or ending `.yaml` — or the name the artefact declares for itself. It is
// required, the positional being mandatory in both forms for a reason that is
// symmetric: the built-in Manifest has no file, so a path can never reach it,
// and `hyper.yaml` declares no name, so a path is the only thing that can (§9,
// ADR-0060).
//
// It carries no `--limit`, because nothing on this screen is a result set: an
// artefact has neither an order nor a cap, and a review that dropped lines
// would be rendering something other than what is about to be approved (§8,
// §9). So `truncated` is the bare `false` on every call.
var reviewTool = tool{
	name:        "review",
	description: "Render §8's review of one artefact — the gutter, the AUTHORITY table and the FLAGS index — as text a human reviewer can be handed verbatim.",
	text:        wholeRendering,
	input: closedObject(`{
		"artefact": {
			"type": "string",
			"minLength": 1,
			"description": "The artefact to review: a repository-relative path — one containing / or ending .yaml — or the name the artefact declares for itself."
		}
	}`, "artefact"),
	output: closedObject(`{
		"rendering": {
			"type": "string",
			"description": "The rendered review surface — the gutter, the AUTHORITY table and the FLAGS index — byte for byte as the command writes it to stdout, and the same string the text block carries. It is on both channels because the rows beside it are what the page is drawn from rather than the page: a caller handed only the rows is handed the ingredients of the thing this tool answers with."
		},
		"rows": {
			"type": "array",
			"items": {
				"oneOf": [
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "kind"],
						"description": "The header row, emitted first: what is being reviewed, and the revision the range opened at. baseline and baseline_absent are the two halves of one member and exactly one of them is written.",
						"properties": {
							"type": {"const": "artefact"},
							"kind": {"enum": ["definition", "procedure", "provider", "target-declaration", "repository-declaration"]},
							"path": {
								"type": "string",
								"description": "The artefact's file, relative to the repository root. Absent on the one artefact with no file in the repository, which ships in the binary."
							},
							"baseline": {"type": "string", "description": "The revision the range opened at, whole: the wire carries no fact to be recognised, so nothing here is abbreviated."},
							"baseline_absent": {
								"enum": ["built-in", "no-store", "not-run", "not-in-clone"],
								"description": "Which of §12's four absences stands where a baseline would: no file at all, nothing to ask, asked and empty, or answered with the object not in this clone."
							},
							`+cadenceGlossMembers+`,
							"last_run": {
								"type": "object",
								"additionalProperties": false,
								"required": ["run", "ended"],
								"description": "The Journal entry the header read an age from. It carries the instant and not the age, an age being a subtraction against the reader's clock.",
								"properties": {
									"run": {"type": "string"},
									"ended": {"type": "string"}
								}
							}
						}
					},
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "line"],
						"description": "One rendered line the gutter has something to say about. The source is not on this list: a review does not decompose into rows, and the consumer already has the file.",
						"properties": {
							"type": {"const": "gutter"},
							"line": {"type": "integer", "minimum": 1},
							"marker": {
								"type": "string",
								"description": "The marker column's text with its alignment padding collapsed to single spaces, and the page's sigils spelled in words. It is one derived fact in one cell rather than a decomposition, a decomposition being a second rendering that can be wrong about the first."
							},
							"changed": {"type": "boolean", "description": "Whether the range touched the line. The revision it is relative to is named in the header, never once per line."}
						}
					},
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "definition", "target"],
						"description": "One pairing of §5's two claims. The four list members are absent where the supply behind them did not load and [] where it loaded and names nothing, which is the distinction the whole table is built on.",
						"properties": {
							"type": {"const": "authority"},
							"definition": {"type": "string"},
							"target": {"type": "string"},
							"definition_kinds": {"type": "array", "items": {"enum": ["read", "mutate", "destroy"]}},
							"target_kinds": {"type": "array", "items": {"enum": ["read", "mutate", "destroy"]}},
							"effective": {"type": "array", "items": {"enum": ["read", "mutate", "destroy"]}},
							"destroy_operations": {"type": "array", "items": {"type": "string"}}
						}
					},
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "flag", "cites_line"],
						"description": "One row of the index into the gutter above. A flag states nothing the gutter does not: it carries no state and no text, and the row on the line it cites is what says which.",
						"properties": {
							"type": {"const": "flag"},
							"flag": {"enum": ["destroy", "opaque", "unbounded", "envelope", "unresolved", "widened", "narrowed", "changed"]},
							"cites_line": {"type": "integer", "minimum": 1, "description": "The line this flag indexes, which is a line a gutter row above marked."},
							"step": {"type": "string", "description": "The Step the flag cites, absent on a flag whose subject is the file rather than a Step."}
						}
					},
					`+problemRow+`
				]
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rendering", "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Artefact string `json:"artefact"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		if err := namesSomething("artefact", named.Artefact, "artefact"); err != nil {
			return nil, err
		}
		// Past one `--`, for providerTool's reason: the positional's name
		// form is matched against namespaces that can hold anything, and its
		// path form against a repository that can hold a file called
		// `--json` (flags.go).
		return []string{"review", "--", named.Artefact}, nil
	},
}

// noArguments is the input schema of a tool that takes none: MCP requires an
// object schema, and this is the closed empty one — no property is declared and
// none is admitted, so `providers({"limit": 10})` is a malformed call rather
// than a cap quietly ignored.
var noArguments = closedObject(`{}`)

// closedObject composes an input or output schema: a closed object over the
// properties given, requiring the names listed.
//
// **`additionalProperties` is false on every schema this surface composes.**
// §9's arguments are closed sets, and a schema that admitted a member it does
// not state would be one under which an override argument is well-formed —
// which is the one thing *no tool takes an override argument of any kind, under
// any name* has to be held by. The same closure on the output is what makes the
// envelope's structured half a contract rather than a description: a member the
// schema does not name is a member this surface does not write.
//
// **The exception is a value whose keys are the artefact's or the world's**, and
// it is an exception to the *composition* rather than to the rule: a Record's
// projected `fields`, a Refusal's `declared` and `observed`, a selector as
// authored. Their members are named by whoever wrote the artefact or by whatever
// answered the call, so a closed object over them would be this surface stating
// a shape it does not own — and stating it wrongly, since the next Manifest
// projects a field this file has never heard of. They are declared open, or
// left untyped where the value may be a scalar as easily as a mapping, which is
// what §8's own wire already carries (§8, ADR-0059).
//
// It is a helper over text rather than a builder over Go values because the
// schemas are the wire's and are read as JSON: what is checked in should be the
// bytes a client receives, not a construction of them. The properties arrive as
// a JSON object literal and the result is compacted, so a schema is legible
// where it is written and one line where it is published.
func closedObject(properties string, required ...string) json.RawMessage {
	// The names are marshalled rather than joined, so that a required
	// property is quoted and escaped by the encoder that will read it back.
	names, err := json.Marshal(required)
	if err != nil {
		panic(err)
	}
	if len(required) == 0 {
		names = []byte("[]")
	}

	schema := fmt.Sprintf(`{"type":"object","additionalProperties":false,"required":%s,"properties":%s}`, names, properties)
	var compacted bytes.Buffer
	// A schema that will not compact is a schema this file mistyped, which
	// is a fact about the source and not about any call: it is raised where
	// the package is initialised rather than carried to a client as a
	// malformed tool.
	if err := json.Compact(&compacted, []byte(schema)); err != nil {
		panic(fmt.Errorf("tool schema %s: %w", properties, err))
	}
	return compacted.Bytes()
}

// namesSomething is the empty-string reading every tool that takes a name or a
// path shares, and the whole of what it holds is that **the server is where a
// schema's claim is made true**: `minLength` is a claim a client may or may not
// check, and an argument that is well-typed and names nothing is a malformed
// call (§9), which is what an error here becomes.
//
// It takes the argument's name and the noun apart, because the two are not one
// word: `operation(provider, operation)` names two namespaces and each message
// has to say which of them was asked, and `check(paths)` has an index to name
// as well. The sentence is composed once so that five arguments cannot come to
// decline in five voices.
func namesSomething(argument, value, noun string) error {
	if value != "" {
		return nil
	}
	return fmt.Errorf("%s is the empty string, which names no %s", argument, noun)
}

// readArguments is the hand-written unmarshalling the low-level registration
// costs, and the whole of what it holds is §9's reading of a bad argument: an
// argument violating a schema is a **malformed call**, so it answers an error
// and the error becomes a JSON-RPC error rather than an envelope with the bit
// set (§9, server.go).
//
// Unknown fields are refused, which is `additionalProperties: false` enforced
// where it is enforceable: a schema is a claim a client may or may not check,
// and this is the reading that makes the claim true on the server. It is the
// same discipline the golden corpus already holds its own fixtures to — a
// misspelt key is a caller asking for something, and answering it as though
// they had not is worse than declining.
//
// An absent `arguments` member is the empty object, which is what a client that
// calls a tool taking no arguments sends: the SDK omits the member rather than
// writing `{}`, and reading nothing as nothing is the same reading.
func readArguments(arguments json.RawMessage, into any) error {
	if len(bytes.TrimSpace(arguments)) == 0 {
		arguments = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	return nil
}

// The Inspection four (§9, issue #199).
//
// **Every row here is its command's, unchanged.** What is new is the arguments
// — §9's typed, closed parameters, and nothing behind them: there is no
// predicate dialect over these tools and none under them, a caller wanting an
// arbitrary filter taking the rows and applying it themselves (ADR-0013) — and
// the truncation marker's second wording, which is the one member of an answer
// §9 spells differently here (envelope.go).
//
// The schemas below are therefore long, and they are long for the reason the
// rows are worth having: a Journal entry read back whole carries five row types
// and a Comparison three, and an `outputSchema` is declared **once and for
// every call of the tool**. What a client is told about a Step under
// `expansion` and about one without it is one schema, because it is one tool.

// provenanceMembers is §7's Provenance as schema members, written once because
// **two rows carry it**: `run_show`'s `provenance` row, at either of the two
// scopes, and the `provenance` member of a `records` row, which carries the
// whole of it under one key (§7, ADR-0043).
//
// It is one fragment for problemRow's reason: there is one declaration behind
// the block on the CLI side (provenance.go) and there should be one schema in
// front of it. Every member follows the ordinary absence rule — a member is
// written at the level where it has one value and omitted from every level
// where it has none — so nothing here is required.
const provenanceMembers = `
	"hyper_version": {"type": "string"},
	"procedure_revision": {"type": "string"},
	"repo_revision": {"type": "string"},
	"repo_dirty": {
		"type": "boolean",
		"description": "Whether the Run read bytes that differ from the revision beside it. Those bytes are nowhere in git, and this is the marker that stops a consumer resolving the revision and believing it read what ran. It is the mark of an artefact that was not committed before it ran, so every revision this Run recorded resolves nowhere and the next review of one of those artefacts opens at no baseline: commit the artefacts before running, and the record points at bytes."
	},
	"definition_revision": {"type": "string"},
	"manifest_digest": {"type": "string"},
	"origin_digest": {"type": "string"}`

// refusalRow is one member of a Refusal on §8's wire, written once because
// **two tools answer it**: `run`, where the Run's own guardrails declined it,
// and `run_show`, reading the same array back out of the entry that recorded
// it (§8, issue #200).
//
// It is one fragment for problemRow's reason, and here the pairing is exact:
// there is one constructor behind the row on the CLI side and both surfaces
// reach it — a Run reports the array and `show` reads it back, one reading of
// one Refusal rather than two that have to agree (refusalRowOf, ADR-0026). The
// `column` a `check` problem carries is the one member absent from it: that
// member rides on `check`'s stream alone and is read back out of no file.
//
// It is a bare object rather than a closedObject for problemRow's reason: what
// it describes is a member of `rows`, and the closure it needs is written into
// it here.
const refusalRow = `{
	"type": "object",
	"additionalProperties": false,
	"required": ["type", "error_code"],
	"description": "One member of a Refusal — one row per problem, and never one row carrying an array. It is the same row as a Run reports it and as the entry recording it is read back. Every member the check did not have is absent rather than written empty.",
	"properties": {
		"type": {"const": "refusal"},
		"error_code": {"type": "string", "description": "One member of §12's closed error_code set, naming the check that declined."},
		"step": {"type": "integer", "minimum": 1, "description": "An artefact coordinate and never an execution fact: the Step it names may have no file in the entry at all, every Refusal that declines before Step 1 citing one that never ran."},
		"step_id": {"type": "string"},
		"operation": {"type": "string"},
		"target": {"type": "string"},
		"declared": {"description": "What the artefact declared, as the value it is: a number, a string, a list or a mapping."},
		"observed": {"description": "What the check found, beside it."},
		"file": {"type": "string"},
		"line": {"type": "integer", "minimum": 1},
		"field": {"type": "string"},
		"message": {"type": "string"},
		"resolved": {
			"type": "object",
			"description": "Each relative operand the citation carries, mapped to the instant it resolved to against this Run's start — not the reader's clock, months later. Its keys are the operands the artefact wrote.",
			"additionalProperties": {"type": "string"}
		}
	}
}`

// remediationRow is §8's `EDIT ONE OF` table on the wire, one row per edit, and
// it is shared by the two tools that carry a Refusal's rows for refusalRow's
// reason: the rows ride between one member and the next, and a schema that said
// one thing under `run` and another under `run_show` would describe two tables
// where the CLI draws one (§8, refusal.go).
const remediationRow = `{
	"type": "object",
	"additionalProperties": false,
	"required": ["type", "file"],
	"description": "One edit past the check above: the coordinate, and either a value to replace or a direction to narrow in. §8 renders these as the EDIT ONE OF table.",
	"properties": {
		"type": {"const": "remediation"},
		"file": {"type": "string"},
		"line": {"type": "integer", "minimum": 1},
		"field": {"type": "string"},
		"from": {"description": "The value the check found, where the replacement is arithmetic."},
		"to": {"description": "The value that would clear it."},
		"hint": {"type": "string", "description": "The direction, where narrowing is a judgement rather than arithmetic."},
		"example_expansion": {"type": "integer", "minimum": 0, "description": "What hyper derived about the hint above: the count a worked example of that narrowing would reach."},
		"resolved": {
			"type": "object",
			"description": "The proposal's relative operands, glossed against the same instant the current value was.",
			"additionalProperties": {"type": "string"}
		}
	}
}`

// provenanceRow is §7's Provenance as a row, at either of its two scopes, and
// it is the members above with the discriminator and the scope around them.
//
// It is shared by the two tools that report a Run's Provenance — `run`, which
// writes it as the Run ends, and `run_show`, which reads it back — and the
// sharing is the point rather than a convenience: what a Run states about the
// code that performed it and what the entry states about the same Run are one
// account (§7, §8, ADR-0043).
const provenanceRow = `{
	"type": "object",
	"additionalProperties": false,
	"required": ["type"],
	"description": "Which code performed the Run, at one of §7's two scopes. Which scope a row is is read off the row itself: a Step's carries step and the Run-wide one does not, and a discriminator beside it would carry that fact twice. Nothing here is abbreviated.",
	"properties": {
		"type": {"const": "provenance"},
		"step": {"type": "integer", "minimum": 1},` + provenanceMembers + `
	}
}`

// cutOrComplete is `truncated` on the three tools whose command orders on an
// axis a `limit` can cut — `runs`, `changes` and `records` — and it states the
// two shapes those commands write (§9, ADR-0102, issue #219).
//
// **Which shape a command writes is a fact about what it ranges over rather
// than a choice** (render.Truncation). §9 fixes three: the bare `false`, the
// bare `true` and the marker object. A command with parameters that narrow its
// axis writes the marker where a limit cut and the bare `false` where nothing
// did; the bare `true` is the namespace listings', which have no axis to name
// and so no marker to compose. So this admits two of the three, and the third
// is absent because these three tools never write it.
//
// The bare boolean alone had stood here, and it was wrong on the one path the
// member exists for: a client running *clients SHOULD validate structured
// results against this schema* over a cut listing was told the server had
// broken its own contract, on the answer §9 says must never look complete.
const cutOrComplete = `{
	"description": "Whether a limit cut this result, in the two shapes these commands write: false where nothing was cut, and the marker naming the cut where something was. There is no cursor behind this stream and no way to ask for the next N — the remedy for a truncated result is a narrower question.",
	"oneOf": [
		{"const": false, "description": "Nothing was cut, and the result is the whole answer to the question that was asked."},
		{
			"type": "object",
			"additionalProperties": false,
			"required": ["axis", "returned", "dropped", "hint"],
			"description": "The cut, named. All four members are written always: they are counts a reader subtracts, and an absent key would read as unknown where the fact is none.",
			"properties": {
				"axis": {
					"enum": ["identity", "time"],
					"description": "§12's closed pair: which of the record's two axes this command orders on, and therefore which one the limit cut."
				},
				"returned": {"type": "integer", "minimum": 1, "description": "How many rows came back."},
				"dropped": {"type": "integer", "minimum": 1, "description": "How many the limit cut. A marker carrying either count at zero would be a truncated result that reads as complete."},
				"hint": {
					"type": "string",
					"description": "The narrower question, naming this tool's arguments where the terminal's marker names its flags — the one wording an answer changes between the two surfaces."
				}
			}
		}
	]
}`

// runsTool carries `hyper runs` — the Journal listed, and the surface that
// enumerates the namespace a `run_id` resolves against.
//
// Its five arguments are the command's five parameters, typed as the flags are
// and named as they are with the hyphens dropped. They are conjunctive, they
// are closed, and there is nothing else: a filter expression here would be the
// predicate dialect ADR-0013 refused, arriving on the surface where it would be
// easiest to write and hardest to review.
//
// **`limit` is offered here where `providers` and `targets` do not offer it**,
// and the difference is what the axis is: this command orders on time and has
// parameters that narrow it, so a cap that cuts hands back a marker naming the
// axis and the arguments that ask a narrower question. A namespace listing has
// neither, which is why its cut is the default's alone (§9, providersTool).
var runsTool = tool{
	name:        "runs",
	text:        locationBeneathSummary,
	description: "List the Journal, newest first: one row per Run, with its Trigger, its outcome, the Targets it bound and the version of hyper that performed it.",
	input: closedObject(`{
		"since": {
			"type": "string",
			"description": "An RFC 3339 instant bounding the window below, inclusive of the instant it names: a timestamp copied off a started member selects the Run it was copied from. There is no relative form and no bare date — those are questions about a clock the caller is not holding."
		},
		"procedure": {"type": "string", "minLength": 1, "description": "A Procedure's name, matched byte-exact over UTF-8."},
		"target": {"type": "string", "minLength": 1, "description": "A Target's name, matched byte-exact over UTF-8. It keeps the Runs that bound it, which is a fact only a Step file carries."},
		"outcome": {
			"enum": ["completed", "refused", "failed"],
			"description": "One member of §12's triple. open is not among them: an entry holding no account of how it ended is in a state and not in the triple, so nothing here selects one."
		},
		"limit": {"type": "integer", "minimum": 1, "description": "The row cap. Omit it for the command's own default; a cut result carries the truncation marker and never a cursor."}
	}`),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "id", "started", "trigger", "procedure", "targets", "hyper_version"],
				"properties": {
					"type": {"const": "run"},
					"id": {"type": "string", "description": "The Run id, whole: what a consumer does with one is hand it back to run_show."},
					"started": {"type": "string"},
					"trigger": {
						"type": "string",
						"description": "The Trigger composed: a clock or a person, which is the whole of what §7 says a Trigger distinguishes. It is on every row, being the only thing that tells a world that has not changed from one nobody has looked at. run_show is where the four facts an executor writes are four members."
					},
					"outcome": {
						"enum": ["completed", "refused", "failed"],
						"description": "Absent on an open entry, the absence carrying that state rather than a fourth value: a started beside no outcome is the whole of what the Store holds about a Run nobody has closed."
					},
					"contested": {
						"type": "boolean",
						"description": "Written where another Run drew an inference beside this entry's own account. It stands beside the outcome rather than inside it, a second account of an entry being no more a fourth value than open is."
					},
					"procedure": {"type": "string"},
					"targets": {
						"type": "array",
						"items": {"type": "string"},
						"description": "The Targets the Run bound, written always and [] where it bound none: every entry carries this fact, and a Refusal that declined before Step 1 bound nothing."
					},
					"hyper_version": {"type": "string"}
				}
			}
		},
		"truncated": `+cutOrComplete+`
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Since     *string `json:"since"`
			Procedure *string `json:"procedure"`
			Target    *string `json:"target"`
			Outcome   *string `json:"outcome"`
			Limit     *int    `json:"limit"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		// The order is §9's own signature order, and each argument is
		// refused empty for namesSomething's reason: `--procedure ""`
		// is a narrowing the command reads as no narrowing at all, so a
		// caller who asked for something would be answered as though
		// they had not.
		narrowed, err := flagsFor(
			namedValue{"since", named.Since, "instant", "--since"},
			namedValue{"procedure", named.Procedure, "Procedure", "--procedure"},
			namedValue{"target", named.Target, "Target", "--target"},
			namedValue{"outcome", named.Outcome, "outcome", "--outcome"},
		)
		if err != nil {
			return nil, err
		}
		argv := append([]string{"runs"}, narrowed...)
		return append(argv, cappedAt(named.Limit)...), nil
	},
}

// runShowTool carries `hyper show <run-id>` — one Journal entry read back
// whole, and **the one tool whose name differs from its command**. A client
// holds every server's tools in one flat namespace, where a bare `show` names
// nothing; the ambiguity the CLI resolved was a different one (§9).
//
// Its two arguments are the command's positional and its one flag. There is no
// `limit`, because the command has none: `show` orders nothing, so there is no
// ordering for a cut to keep the first N of and no axis for a marker to name —
// and what a cap would do instead is hand back a Run's account with its last
// Steps dropped, which is the partial answer wearing a complete one's shape §9
// forbids (show.go).
//
// **`expansion` is a boolean and never a mode something else turns on**
// (ADR-0013). Under it each Step row carries the `selector` it was expanded
// from, what it expanded to and the Bound it was read against; without it none
// does, an Expansion of five hundred members being the whole of what a Step
// reached and almost never what a reader of a Disposition came for.
//
// **A `run_id` the Store lacks is a JSON-RPC error**: it satisfies every schema
// and still names nothing, which is §9's third member of the malformed set —
// and a partial id resolves to nothing anywhere, so a prefix and a typo arrive
// at one message (§9, ADR-0047, ADR-0060).
//
// §9's sketch named a `disposition` row here, carrying `state` and `failed_path`
// with the Expansion split off into rows of its own. The command has written
// `entry`, `refusal`, `remediation`, `provenance` and `step` rows since
// milestone 8, and those are what this answers: §12's opening rule sends one
// fact to two wires under one name, and a second shape for one entry is where
// the day comes that the two surfaces disagree about what a Run did (§9, §12,
// ADR-0026, issue #199).
var runShowTool = tool{
	name:        "run_show",
	description: "Read one Journal entry back whole: its header, the Refusal its Run recorded, and one row per Step record with that Step's own Provenance beside it.",
	input: closedObject(`{
		"run_id": {
			"type": "string",
			"minLength": 1,
			"description": "The Run id, whole. Nothing anywhere resolves a partial one, so a prefix names nothing exactly as a typo does; hyper runs enumerates the namespace it resolves against."
		},
		"expansion": {
			"type": "boolean",
			"description": "Whether each Step row carries the selector it was expanded from, what it expanded to and the Bound it was read against. It is the destination §8's bound-exceeded page points a reader at."
		}
	}`, "run_id"),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"oneOf": [
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "run_id", "procedure", "trigger", "started_at", "dry_run"],
						"description": "The entry's header, emitted first: what its own run.json holds, and the account or accounts it carries of how it ended. The four states §7 classifies an entry into are read off which of outcome and closed_by stand.",
						"properties": {
							"type": {"const": "entry"},
							"run_id": {"type": "string"},
							"procedure": {"type": "string"},
							"trigger": {
								"type": "object",
								"additionalProperties": false,
								"required": ["cause", "executor"],
								"description": "The Trigger as the entry holds it: a mapping and never a composed string, four facts whose shape differs by executor not packing into one without a grammar and a parser.",
								"properties": {
									"cause": {"type": "string"},
									"executor": {"type": "string"},
									"actor": {"type": "string"},
									"host": {"type": "string"},
									"run_id": {"type": "string", "description": "The executor's own run id, which is not hyper's: the entry's is the member of the same name on the row above."},
									"run_attempt": {"type": "integer", "minimum": 1},
									"job_url": {"type": "string"}
								}
							},
							"started_at": {"type": "string"},
							"dry_run": {
								"type": "boolean",
								"description": "Written always, false included — §7's one exception to the absence rule, because what a reader that takes its absence for false gets wrong is unrecoverable."
							},
							"outcome": {"enum": ["completed", "refused", "failed"]},
							"ended_at": {
								"type": "string",
								"description": "The owner's and never a closer's: a closing write's instant is on the closing Run's clock, and putting it here would invite a cross-entry subtraction §7 forbids."
							},
							"closed_by": {
								"type": "array",
								"description": "Every inference another Run drew about this entry, one member per closing write. Both this and outcome standing is a contest, which hyper reports and never decides.",
								"items": {
									"type": "object",
									"additionalProperties": false,
									"required": ["run_id", "outcome", "ended_at"],
									"properties": {
										"run_id": {"type": "string", "description": "The closing Run, which is the file's name rather than one of its members."},
										"outcome": {"const": "failed", "description": "What §7 fixes every closing write as. It is written rather than left implied because the page states it."},
										"step": {"type": "integer", "minimum": 1},
										"ended_at": {"type": "string"}
									}
								}
							}
						}
					},
					`+refusalRow+`,
					`+remediationRow+`,
					`+provenanceRow+`,
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "step", "disposition"],
						"description": "One Step record the entry holds, in the Run's own written order. A Step the Run never reached wrote no file, so it has no row here and no provenance row either: that Disposition is read from a silence inside a closed entry.",
						"properties": {
							"type": {"const": "step"},
							"step": {"type": "integer", "minimum": 1},
							"id": {"type": "string"},
							"path": {"type": "string", "description": "The invocation chain where the Step was reached through a nested Procedure."},
							"definition": {"type": "string"},
							"operation": {"type": "string"},
							"provider": {"type": "string"},
							"target": {"type": "string"},
							"kind": {"enum": ["read", "mutate", "destroy"]},
							"disposition": {"type": "string", "description": "What became of the Step, as §12 names it."},
							"started_at": {"type": "string", "description": "Absent on a record a reaper wrote: a closing write does not know when the Step began, which is an honest absence rather than the year 1."},
							"ended_at": {"type": "string"},
							"records": {
								"type": "array",
								"items": {"type": "string"},
								"description": "The identities this Step concluded about — the members, and not the count §8's Step table renders. Absent where the Disposition carries no set at all, and [] where a Step ran and its Expansion resolved to nothing; the two are different answers."
							},
							"unchanged_since": {
								"type": "string",
								"description": "The Run the members above were read from, where this entry holds a digest and no members of its own. It is what keeps show from presenting another entry's bytes as though they were this one's."
							},
							"selector": {
								"type": "object",
								"additionalProperties": false,
								"required": ["declared", "expanded_to"],
								"description": "Written under expansion and nowhere else.",
								"properties": {
									"declared": {"description": "The selector as authored, in the canonical bytes the entry holds for it."},
									"expanded_to": {
										"type": "array",
										"items": {"type": "string"},
										"description": "What it expanded to, in Expansion order and never sorted: on a serial destroy the halt point is legible by position and nowhere else. Written whenever a selector exists, the empty list included."
									},
									"bound": {"type": "integer", "minimum": 0}
								}
							},
							"resolved": {
								"type": "object",
								"description": "Each relative operand the selector carries, mapped to the instant it resolved to against this Run's start. It rides with selector and under expansion alone.",
								"additionalProperties": {"type": "string"}
							},
							"pattern": {
								"type": "object",
								"additionalProperties": false,
								"description": "hyper's own account of the work, supplied by no Provider: what makes a Step that took four minutes legible as four minutes of something.",
								"properties": {
									"attempts": {"type": "integer", "minimum": 1},
									"pages": {"type": "integer", "minimum": 1},
									"polls": {"type": "integer", "minimum": 1}
								}
							},
							"answered": {
								"type": "array",
								"description": "One entry per member of this Step's Expansion whose effectful call did not give the ordinary answer, in Expansion order. A read's status is the answer and is in the Record; which of the three cases ended the Step is read from the disposition beside this list rather than off any one entry.",
								"items": {
									"type": "object",
									"additionalProperties": false,
									"properties": {
										"member": {"type": "string", "description": "The member of expanded_to this answer is about, and absent where the Step resolved no selector and there is none to name. It is what tells which of the Tombstones a destroy wrote was written on a 404."},
										"host": {"type": "string"},
										"status": {"type": "integer", "description": "Absent where no response arrived at all, which is the member whose request provably never left."},
										"command": {"type": "string"},
										"exit_code": {"type": "integer", "description": "Absent where the command was never started: 0 is an answer a shell command gives, and reading an unset field as one is how a request that never left acquires an exit code."}
									}
								}
							},
							"projection_failed_path": {"type": "string", "description": "§6's projection failure alone, and the records above are then partial."}
						}
					}
				]
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			RunID     string `json:"run_id"`
			Expansion bool   `json:"expansion"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		if err := namesSomething("run_id", named.RunID, "Run"); err != nil {
			return nil, err
		}
		argv := []string{"show"}
		if named.Expansion {
			// Before the `--`, because that is where the command
			// takes it: `--expansion` comes off the line before the
			// shared parser sees it and stops at the first `--`
			// (show.go).
			argv = append(argv, "--expansion")
		}
		// Past one `--`, for providerTool's reason: a Run id is matched
		// byte-exact against a namespace, and the parser already states
		// that "--" ends the flags (flags.go).
		return append(argv, "--", named.RunID), nil
	},
}

// Execution (§9, issue #200).
//
// **The tool that closes the loop.** An agent that cannot run also cannot read
// back the Record it just caused, which is what this surface exists for. §9
// puts the effectful half here deliberately: restricting the surface to
// authoring and reads would make *who is calling* an axis of authority, and no
// guardrail §5 states is a function of that — an unattended Run on a Cadence is
// already accepted, and a call made by an agent with a human watching it is
// strictly safer than that.

// runTool carries `hyper run <procedure>` — the one tool of the thirteen that
// writes to the record on its own account, and the one whose answer carries
// §12's outcome triple.
//
// **`run` takes a Procedure and nothing else**, as the command does. Every Run
// is a Run of a Procedure (ADR-0036), so there is no single-Operation arm of
// this tool and a call carrying a `definition` is an argument violating a
// schema — a protocol error, which is what this surface has in place of exit
// `2`. There is no `target` either: a Procedure declares its own envelope, so
// one supplied here is redundant with the artefact or it is authority arriving
// after review (§5, ADR-0008).
//
// **The positional is a name and takes no second form**, where `reviewTool`'s
// one screen up takes a path beside one. That is ADR-0090's, and this schema
// asserts it rather than restating it: the two forms there are two namespaces
// and here there would be one namespace twice. A caller holding
// `procedures/deploy.yaml` needs no second form either, `name-mismatch` pinning
// a name to its file's basename (§4) — which is why the description says so.
//
// **There is no `inputs` argument.** A Procedure is fully bound by its
// artefact, and a value supplied at call time is Step behaviour appearing on no
// reviewed line — authority arriving after review, which is the shape ADR-0008
// removed and the same shape as the `--force` that is absent everywhere else.
// The schema is closed, so every one of those names is refused by the schema
// rather than by a check written here (closedObject).
//
// **A `dry_run` call is answered in both halves**, and the `step` row's
// `withheld` is the half that used to be missing. An agent calling it asks
// *what would this do, and where does it stop*, and where it stopped was a
// page fact: the withheld Step's Disposition is *never reached*, which is the
// one it shares with every Step behind it. The member is on the row, so this
// surface reads the boundary of a partial answer rather than inferring it from
// `outcome` and `dry_run` — an inference a Run the world resisted breaks (§8,
// ADR-0091, issue #206). No composition here changes: the rows are §8's, one
// renderer writes both forms, and this schema declares what that renderer now
// writes (ADR-0026, envelope.go).
//
// **`secret_sink` is the CLI's `--secret-out` under the name of the thing it
// supplies**, a flag named for a direction having no direction to name in an
// argument object. It is chosen by the caller and **never defaulted by
// `hyper`**: a sink supplied automatically would make `hyper` a place a secret
// lives, and would delete the guardrail an absent one earns once the file is
// written at all (ADR-0007). Everything the CLI states about the path holds
// whoever named it, and both faults it can have — `-`, and a path inside the
// repository working tree — are the usage errors the command already makes
// them, arriving here as protocol errors (§9, run.go, ADR-0060).
//
// **No sink is written by this hyper, and this surface says so by Refusing.** A
// Run reaching a Step whose Operation declares secret output declines whether a
// path was named or not, and the Refusal arrives here as any other does
// (`secret-sink-unwritten`, §12, ADR-0146). An argument accepted, described and
// silently inert is the one thing an agent has no way to discover (issue #266).
//
// **Returning the secret in the tool result is not one of the sink's forms**,
// and nothing here could make it one: what the sink names is a path, and a Run
// will write the file. A generated credential in a tool result is a credential
// in an agent's context and from there in whatever transcript that agent
// writes.
//
// **The call is synchronous and there is no handle in this schema to poll.**
// `run` returning an id the caller comes back for would invent a Run that
// outlives its caller with nothing watching it, which is a daemon with extra
// steps — so what a caller holds while the Run works is the progress
// notification at each Step boundary, and what it holds when the call returns
// is the whole answer (§9, ADR-0092, server.go). The output schema is closed
// over §9's five members, so a handle is not a thing this tool could grow
// quietly.
//
// **A cancelled call drains.** The client cancels the request, the SDK cancels
// the handler's context, and that is the drain §6 already states: the Step in
// flight finishes, no further Step starts, and the Run closes its own entry
// `failed`. A client that gives up entirely gets no delivery at all and needs
// no machinery of its own — the stdio server dies with it, and the open entry
// is closed by the next invocation with the Step in flight recorded *attempted,
// outcome unknown* (§6, §7, ADR-0015, ADR-0092).
//
// It carries no `--limit`, its command carrying none: a Run reports what it
// just did rather than ranging over a namespace, so there is no result set for
// a cap to cut and `truncated` is `null` on every call (§9, envelope.go).
//
// **Both sink faults come back naming `--secret-out` and not `secret_sink`**,
// and that is the rule holding rather than a rough edge: a usage error's
// message is what the command wrote where the CLI writes a human sentence, and
// this surface forwards it so an agent reads the sentence a person would have
// read (§9, issue #196). §9 spells one wording differently between the two
// surfaces and one only — the truncation marker's hint — so a rewrite here
// would be this package holding an opinion about a sentence addressed to a
// caller (envelope.go, render.Narrowing).
var runTool = tool{
	name:        "run",
	description: "Run one Procedure through every guardrail §5 states, writing the Records, the Journal entry and the Provenance the command writes. Refuses rather than overriding: no argument here is a bypass.",
	input: closedObject(`{
		"procedure": {
			"type": "string",
			"minLength": 1,
			"description": "The Procedure to run: the name the artefact declares for itself, matched byte-exact over UTF-8. It is a name and never a path — name-mismatch pins a name to its file's basename, so the Procedure in procedures/deploy.yaml is run as deploy — and it carries no Target: a Procedure is fully bound and declares its own Target envelope."
		},
		"dry_run": {
			"type": "boolean",
			"description": "Perform the reads this Run reaches and stop rather than simulating an effect. A rehearsal writes a Journal entry marked as one, names the Step it withheld with every Step after it never reached, and reports completed — a halted rehearsal is the correct outcome of a correct operation."
		},
		"secret_sink": {
			"type": "string",
			"minLength": 1,
			"description": "Where a Step declaring secret output would write it: an absolute path outside the repository working tree. It is never defaulted, and the secret is never returned in this result. Nothing writes the file yet, so supplying a path rescues no Run: this version accepts it and refuses its faults, and a Run reaching such a Step Refuses under secret-sink-unwritten whether one was named or not."
		}
	}`, "procedure"),
	output: closedObject(`{
		"outcome": {
			"enum": ["completed", "refused", "failed"],
			"description": "§12's triple, and this tool's alone among the thirteen. It is the discriminator and isError is not: one bit cannot carry three states. failed is the one a caller may retry — past it lies time, where past a refused lies an act of somebody's."
		},
		"run_id": {
			"type": "string",
			"description": "The entry this Run wrote, whole. Absent exactly where no entry was written: the version pin gate, the bootstrap store-absent, and a Run that lost the Store before it was attempted."
		},
		"dry_run": {
			"type": "boolean",
			"description": "Whether this was a rehearsal. Written always, false included — §7's one exception to the absence rule, because what a reader that takes its absence for false gets wrong is unrecoverable."
		},
		"rows": {
			"type": "array",
			"items": {
				"oneOf": [
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "step", "id", "kind", "disposition"],
						"description": "One Step that reached a Disposition, in the order the Run ran them. What each Record did is the Comparison's rendering rather than the Run's, and changes is what emits it.",
						"properties": {
							"type": {"const": "step"},
							"step": {"type": "integer", "minimum": 1},
							"id": {"type": "string"},
							"path": {"type": "string", "description": "The invocation chain a Step reached through a nested Procedure was reached under, absent on a top-level Step."},
							"kind": {"enum": ["read", "mutate", "destroy"]},
							"disposition": {"type": "string", "description": "What became of the Step, as §12 names it."},
							"records": {
								"type": "integer",
								"minimum": 0,
								"description": "How many identities the Step concluded about, which is not how many it wrote. Absent where the Disposition carries no set at all, and 0 where a Step ran and its Expansion resolved to nothing; the two are different answers."
							},
							"expanded": {
								"type": "integer",
								"minimum": 0,
								"description": "What the Expansion resolved to, where the Step stopped short of it and the count above accounts for only part. Which Records those are is run_show under expansion and nowhere else."
							},
							"withheld": {
								"const": true,
								"description": "The Step a rehearsal stopped at, written true on that one Step and absent on every other and on every Run that is not a rehearsal. It is the boundary of a partial answer: this Step's effect was withheld rather than simulated, and the Steps after it are never-reached behind it. Do not read the first never-reached row as this — a Run the world resisted leaves those rows too."
							}
						}
					},
					`+refusalRow+`,
					`+remediationRow+`,
					`+provenanceRow+`
				]
			}
		},
		"truncated": {
			"type": "null",
			"description": "A Run reports what it just did rather than ranging over a namespace, so there is no result set for a limit to have cut and no marker to carry."
		}
	}`, "outcome", "dry_run", "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		named, err := runArguments(arguments)
		if err != nil {
			return nil, err
		}
		// Both flags come off the command's line **before** the shared
		// parser sees it and both stop at the first `--`, so they go
		// ahead of the positional here — which is what keeps a Procedure
		// named `--dry-run` runnable (run.go).
		argv := []string{"run"}
		if named.DryRun {
			argv = append(argv, "--dry-run")
		}
		// The sink through namedValue, which is the reading every
		// optional argument on this surface already gets: absent is the
		// flag left off, and the empty string is a caller who asked for
		// a sink and named none — a malformed call, and never the
		// absence that makes a Step declaring secret output Refuse (§9,
		// ADR-0007).
		sink, err := flagsFor(namedValue{argument: "secret_sink", value: named.SecretSink, noun: "path", flag: "--secret-out"})
		if err != nil {
			return nil, err
		}
		argv = append(argv, sink...)
		// Past one `--`, for providerTool's reason: the positional's
		// name form is matched against a namespace that can hold
		// anything, and its path form against a repository that can
		// hold a file called `--json` (flags.go).
		return append(argv, "--", named.Procedure), nil
	},
	executes: func(arguments json.RawMessage) (execution, error) {
		named, err := runArguments(arguments)
		if err != nil {
			return execution{}, err
		}
		return execution{dryRun: named.DryRun}, nil
	},
}

// runCall is one `run` call's arguments, read.
//
// `secret_sink` is a pointer for namedValue's reason: **absent and empty are
// two different calls**. A caller who sent `""` asked for a sink and named
// none, and reading that as *no sink* would answer a malformed call with
// whatever a call that named nothing gets (§9, ADR-0007).
//
// The distinction is held here rather than derived from what the sink then
// does, and it outlives the sink's own state: no hyper writes the file, so a
// Run reaching a Step that declares secret output Refuses whether one was
// named or not (`secret-sink-unwritten`, §12, ADR-0146), and the day the
// format lands an absent sink is a Refusal of its own again (issue #266).
type runCall struct {
	Procedure  string  `json:"procedure"`
	DryRun     bool    `json:"dry_run"`
	SecretSink *string `json:"secret_sink"`
}

// runArguments is the one reading of a `run` call's arguments, and it is asked
// twice: once for the command line, and once for what the envelope needs to say
// about a Run that never reached a row (runTool).
//
// **It is one function rather than two readings** for the reason the golden
// corpus reads a case's argv once: a call that answered the question
// differently depending on who asked would be one whose command line said one
// thing and whose envelope said another.
//
// The positional's empty string is refused here and the sink's is refused where
// the flag is built, which is namedValue's own reading — the sink is optional
// and the Procedure is not, so only one of the two has an absence to tell an
// empty string from (runTool.argv).
func runArguments(arguments json.RawMessage) (runCall, error) {
	var named runCall
	if err := readArguments(arguments, &named); err != nil {
		return runCall{}, err
	}
	if err := namesSomething("procedure", named.Procedure, "Procedure"); err != nil {
		return runCall{}, err
	}
	return named, nil
}

// probeTool carries `hyper probe <provider> <operation>` — §9's second
// Execution tool, and the one that protects the review surface (issue #201).
//
// **The reason this surface needs a Probe is not agent convenience.** Writing a
// file is cheap for an agent; what is not cheap is §8's review model, which
// dies by volume — an agent authoring a Manifest against an unfamiliar API asks
// *what does this endpoint actually answer* twenty times, and if every one of
// those questions costs a reviewed Definition then the set of Definitions stops
// being something a human reads and the oversight story goes with it (§9).
//
// **It carries no `outcome` key**, and the table is where that is declared: a
// Probe writes no Record and no Journal entry, so it has no outcome triple to
// report, and a tool leaving `executes` nil is one whose envelope carries none
// (§9, ADR-0009, envelope.go). What ends its rows is `result` and never
// `outcome`, which is the command's own terminal row lifted like every other
// (§8, ADR-0026).
//
// **Everything else a Probe declines, it declines in the command.** The opaque
// Operation, the effectful one, the input the Operation does not declare, the
// declared input left out and the value that will not read as its declared type
// are the command's own usage errors, and each arrives here as the JSON-RPC
// error §9 answers a malformed call with, carrying no `error_code` — a code
// naming a check that declined an **artefact**, and a value supplied at a call
// not being one (§9, ADR-0060). A host outside what the Target named `local`
// grants is the other half and the other shape: that one is a guardrail
// declining, so it comes back `isError: true` with the Refusal rendered whole.
var probeTool = tool{
	name:        "probe",
	description: "Invoke a read Operation against local without a Definition, and answer the projection beside the raw response. It writes no Record and no Journal entry: a throwaway question costs no reviewed artefact. Supply `response` instead and no call is made at all: hyper reads the Operation's record: block against a response object you fetched yourself, which is how the projection of an Operation a Probe may not invoke — an effectful one, or one behind a credential — is checked without spending a Run.",
	input: closedObject(`{
		"provider": {"type": "string", "minLength": 1, "description": "The Provider's name, as its Manifest declares it."},
		"operation": {"type": "string", "minLength": 1, "description": "The Operation's name, as a key of that Manifest's own operations: block. Without a supplied response it declares kind: read and is not opaque — a Probe may invoke neither an effectful Operation nor an opaque one, whatever any Target grants. With one, no Operation is out of reach, both rules bounding a request that is not made."},
		"response": {
			"type": "string",
			"minLength": 1,
			"description": "A repository-relative path to a JSON file holding the response object §12 closes for the Operation's Capability — its members, spelled as they are after a call. Supplying one makes no call at all: hyper resolves no credential, reaches no host, and answers only the projection the Operation's record: block derives from what you handed it, beside the paths that resolved to nothing. Every rule a Probe otherwise carries bounds a request leaving this machine, so with no request the Operation may be of any Kind and may be opaque — which is how a mutate's projection, the response to a create, is reachable at all. The path is read against the repository like every other path argument."
		},
		"inputs": {
			"type": "object",
			"description": "The Operation's inputs, one object keyed by input name, in place of the CLI's repeated --input. Every declared input is supplied and no other name is: there is no null and no key-omission syntax, so an input left out has no sink to render at. Beside a supplied response there is no request and so no sink, and inputs are optional there: what one still reaches is an identity: written as a template hole rather than a path. Each value is read against the type the Operation declares at that position rather than by what the value looks like, so a string carrying digits and a number are one value at an integer position; object, array and null read as nothing anywhere and are refused here.",
			"propertyNames": {
				"minLength": 1,
				"pattern": "^[^=]*$",
				"description": "An input's name, as the Operation's own inputs: block declares it. It carries no = : the command line this tool builds spells an input as one --input name=value pair and splits it at the first =, so a name carrying one names an input no pair can address."
			},
			"additionalProperties": {"type": ["string", "number", "boolean"]}
		}
	}`, "provider", "operation"),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "provider", "operation", "supplied", "projection", "unresolved", "response"],
				"properties": {
					"type": {"const": "probe_result"},
					"provider": {"type": "string"},
					"operation": {"type": "string"},
					"supplied": {
						"type": "boolean",
						"description": "Whether the response beneath was handed to hyper rather than fetched by it. A response hyper called for and a response a caller supplied are two different claims about the world, and the row says which one this is."
					},
					"projection": {
						"type": "array",
						"description": "What hyper derived from the response, in the shape the Records would have held. One entry under an Operation of one cardinality and one per member of the collection over: named under series — and [] where over: itself resolved to nothing, which is named in unresolved rather than reported as an error.",
						"items": {
							"type": "object",
							"additionalProperties": false,
							"required": ["fields"],
							"properties": {
								"identity": {"description": "What the Record's identity: resolved to, and absent where it resolved to nothing."},
								"fields": {"type": "object", "description": "The fields that resolved, keyed by the names the Manifest records them under, in the Manifest's own order. {} where every one of them resolved to nothing, which is an absence a reader reads rather than an error hyper reports."}
							}
						}
					},
					"unresolved": {
						"type": "array",
						"description": "Every path the Operation's record: block authored that resolved to nothing, named once however many members it failed against. This is the half a Run has nowhere to put: a field whose path resolved to nothing is simply absent from a version, and an author reading a projection cannot tell that from a field never declared.",
						"items": {
							"type": "object",
							"additionalProperties": false,
							"required": ["position", "path"],
							"properties": {
								"position": {"type": "string", "description": "Where the path is authored: over: or identity: with their colon, being keys of the record: block, or the name a field is recorded under."},
								"path": {"type": "string", "description": "The path written there, as the Manifest writes it."}
							}
						}
					},
					"response": {
						"type": "object",
						"description": "The raw response beside the projection, which no credentialled surface shows: a Probe that called binds local, which carries no credential slot, so the wire is visible by construction rather than by a flag, and a Probe that did not call is echoing bytes the caller already holds. A host that answered nothing at all is this object carrying host and nothing else."
					}
				}
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Provider  string                     `json:"provider"`
			Operation string                     `json:"operation"`
			Response  string                     `json:"response"`
			Inputs    map[string]json.RawMessage `json:"inputs"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		// The two positionals are refused separately for operationTool's
		// reason read across: a Probe resolves them against the Provider
		// set exactly as `operation` does, and the two namespaces are two
		// (§9, probe.go).
		if err := namesSomething("provider", named.Provider, "Provider"); err != nil {
			return nil, err
		}
		if err := namesSomething("operation", named.Operation, "Operation"); err != nil {
			return nil, err
		}
		supplied, err := suppliedInputs(named.Inputs)
		if err != nil {
			return nil, err
		}
		// The inputs go ahead of the `--` and the two names go past it.
		// The command takes its repeated flag off the argument list
		// before the shared parser sees it and stops at the first `--`,
		// so a Provider or an Operation spelled like a flag is still the
		// positional it is (flags.go, probe.go).
		argv := append([]string{"probe"}, supplied...)
		if named.Response != "" {
			argv = append(argv, "--response", named.Response)
		}
		return append(argv, "--", named.Provider, named.Operation), nil
	},
}

// suppliedInputs is the `inputs` object as the command line carries it: one
// `--input name=value` pair per member.
//
// **They go over sorted by name**, which is not the order a caller wrote them
// in because a JSON object has no order to read. Two clients spelling one call
// with their keys two ways would otherwise build two command lines and, where
// both name an input the Operation does not declare, be told about their faults
// in two orders. The command keeps the order it was typed in for a reason that
// does not reach here — a repeated flag can name one input twice and an object
// cannot (probe.go).
func suppliedInputs(inputs map[string]json.RawMessage) ([]string, error) {
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	slices.Sort(names)

	argv := make([]string, 0, 2*len(names))
	for _, name := range names {
		// The schema's `propertyNames` made true, which is where the two
		// halves of it belong: `minLength` and `pattern` are claims a
		// client may or may not check, and the server is where a claim
		// becomes a refusal — the same reading `namesSomething` takes
		// over a value and `readArguments` takes over the closure
		// (probeTool).
		if name == "" {
			return nil, errors.New("inputs names an input with the empty string, which names no input")
		}
		if strings.Contains(name, "=") {
			return nil, fmt.Errorf("inputs %s: an input name carrying an = cannot be spelled as one --input pair, which is the command line this tool builds", name)
		}
		value, err := inputText(name, inputs[name])
		if err != nil {
			return nil, err
		}
		argv = append(argv, "--input", name+"="+value)
	}
	return argv, nil
}

// inputText is one input's value as the pair carries it: the JSON scalar
// written out as text.
//
// **The spelling crosses and the typing does not.** A value is read against the
// type the Operation **declares** at that position rather than by what the value
// looks like (ADR-0081), and that declaration is in a Manifest this file has
// never seen — so what a number goes over as is the digits the caller wrote,
// and a `1.0` at an `integer` position is declined by the command in its own
// words rather than quietly rounded here. A tool that read the value by its JSON
// type would be this surface typing it by what it looks like, which is the one
// reading ADR-0081 removed.
//
// **The three JSON scalars cross and nothing else does.** §12's types are all
// scalars, and `object` and `array` read as nothing at every position a hole
// fills (ADR-0078), so a member that is one of those — or a `null`, which names
// no value at all — is a well-typed argument naming a value no input can hold.
// That is a malformed call and arrives as a protocol error, carrying no
// `error_code`: nothing was reviewed, so no check declined (§9, ADR-0060).
func inputText(name string, value json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(value))
	// The digits as they were written. Through a float64 an integer too
	// large to hold would arrive rounded and a `1.0` would arrive as `1`,
	// either of which is this surface editing a value on its way to a schema
	// that has not read it yet — and the second is the one that matters,
	// since what an `integer` position refuses is exactly that spelling.
	decoder.UseNumber()
	var read any
	if err := decoder.Decode(&read); err != nil {
		return "", fmt.Errorf("inputs %s: %w", name, err)
	}
	switch held := read.(type) {
	case string:
		return held, nil
	case json.Number:
		return held.String(), nil
	case bool:
		return strconv.FormatBool(held), nil
	}
	return "", fmt.Errorf("inputs %s: %s is no value an input carries; an input holds one scalar — a string, a number or a boolean", name, value)
}

// changesTool carries `hyper changes [procedure]` — §8's Comparison, and the
// question this whole surface exists to close the loop on: *what differs from
// when we last looked*.
//
// **`procedure` is the command's positional and not a filter**, which is the
// one difference between this argument set and `runs`': naming one selects the
// rendering, and naming none compares across every Procedure at once (§9,
// changes.go).
//
// **`record_kind` is the CLI's `--kind` spelled out.** In a flat argument object
// beside tools carrying an Operation's Kind, one name cannot hold two senses:
// a **Kind** is `read`, `mutate` or `destroy`, and this is `asset` or
// `observation`. §9 writes the argument out this way and the CLI's own flag
// value is documented under the same reading (flags.go).
//
// **`since`, `between` and `subject` name one window three ways, and naming it
// more than one way is a JSON-RPC error.** So is a `between` naming a rehearsal
// as its baseline, an open entry, one Run twice, two Procedures, or the two ends
// the wrong way round: every one of them is a well-typed argument that names no
// window this surface can render, and the command's own sentence is what comes
// back (§9, ADR-0060).
//
// **`subject` is the argument this whole tool is reachable from a rehearsal
// by.** Between a `--dry-run` and the effecting Run that follows it, the
// Observations the rehearsal recorded are in the Store and nothing rendered
// their values: `records` finds a version and does not read one, `run_show`
// renders Dispositions, and the Comparison had no subject to take. Naming the
// rehearsal is what gives it one, and the answer is what that Run read rather
// than a claim about what the world became (§8, ADR-0115, issue #235).
var changesTool = tool{
	name:        "changes",
	description: "Report what changed between two Runs of a Procedure: the Assets you moved, the Observations the world moved, and what moved in the code between them.",
	input: closedObject(`{
		"procedure": {
			"type": "string",
			"minLength": 1,
			"description": "The Procedure to compare Runs of. Omit it to compare across every Procedure the Journal holds, one block each — which is a fold and never a sum: there is no grand total across windows with different baselines."
		},
		"since": {"type": "string", "description": "An RFC 3339 instant bounding the window, inclusive of the instant it names. It is one of two ways of naming a window and naming it both ways at once is a malformed call."},
		"between": {
			"type": "array",
			"items": {"type": "string", "minLength": 1},
			"minItems": 2,
			"maxItems": 2,
			"description": "The window's two ends named directly, baseline first and subject second — the order the header renders them in. A pair given the other way round is refused rather than quietly reordered."
		},
		"subject": {"type": "string", "minLength": 1, "description": "The window's subject named by id, with the baseline derived behind it: the Run before it, skipping rehearsals and open entries. It is the one place a rehearsal may be a side of a window, and it is what renders the field values a --dry-run recorded before any effectful Run exists. A rehearsal is still never a baseline."},
		"target": {"type": "string", "minLength": 1, "description": "A Target's name, matched byte-exact over UTF-8. It narrows the two Record tables and never the header or the code facts beneath them."},
		"record_kind": {
			"enum": ["asset", "observation"],
			"description": "One of §7's two Record types, which is the split between the two Record tables. It is the CLI's --kind under a name that cannot be read as an Operation's Kind."
		},
		"limit": {"type": "integer", "minimum": 1, "description": "The cap on the two Record tables' rows. It cuts neither the window rows nor the code facts: narrowing what a reader looked at may not narrow what they are told changed."}
	}`),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"oneOf": [
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "procedure", "subject"],
						"description": "One window's header, and the row every block opens with. baseline is absent where there is none, which a window has exactly one way of: the subject is the first Run of its Procedure.",
						"properties": {
							"type": {"const": "window"},
							"procedure": {"type": "string"},
							"baseline": `+windowSide+`,
							"subject": `+windowSide+`
						}
					},
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "change", "target", "definition", "name", "fields"],
						"description": "One Record that moved. The type is which of the two tables holds it — asset is YOU DID THIS and observation is THE WORLD MOVED — the split being by actor rather than by column.",
						"properties": {
							"type": {"enum": ["asset", "observation"]},
							"change": {"type": "string", "description": "What happened to it between the two ends, as §8 names it."},
							"target": {"type": "string"},
							"definition": {"type": "string"},
							"name": {"type": "string"},
							"from_ordinal": {"type": "integer", "minimum": 1, "description": "Absent where the end has no version to name, which is what appeared and vanished are."},
							"to_ordinal": {"type": "integer", "minimum": 1},
							"confirmed_at": {"type": "string", "description": "When a destruction was confirmed, on a destroyed row alone."},
							"fields": {
								"type": "object",
								"description": "What the projection held, every value whole: a pair of values where the field changed, and the value alone where one end held nothing. Its keys are the Record's own fields, so it is the one object here whose members hyper does not state. Written always, the empty mapping included — an empty one is what says hyper destroyed this and never observed what it was."
							}
						}
					},
					{
						"type": "object",
						"additionalProperties": false,
						"required": ["type", "fact"],
						"description": "One code fact, or the catch-all that terminates the table. The two are told apart by fact. subject_kind and subject stand together where the fact has an artefact subject and neither stands where it does not — repo_revision belongs to no artefact a reader can open.",
						"properties": {
							"type": {"const": "code"},
							"subject_kind": {"type": "string"},
							"subject": {"type": "string"},
							"fact": {"type": "string"},
							"from": {"description": "The baseline's value, as the value it is."},
							"to": {"description": "The subject's value."},
							"from_phrase": {"type": "string", "description": "A Cadence expression glossed, where the fact is one: the gloss's parts ride beside the expression rather than as a composed line."},
							"to_phrase": {"type": "string"},
							"from_rate": {"type": "number"},
							"to_rate": {"type": "number"},
							"count": {"type": "integer", "minimum": 0, "description": "The catch-all's own: how many moved lines no classed row above reports. Zero is a count, which is why the table can read 0 other lines changed and still carry this row."},
							"baseline_absent": {"type": "string", "description": "Stands in place of count where the bytes could not be read: the object is not in this clone."},
							"command": {"type": "string", "description": "The git diff a reader runs, abbreviated as the page draws it — a command rather than an id, and one git resolves short. Absent where either side recorded repo_dirty."}
						}
					}
				]
			}
		},
		"truncated": `+cutOrComplete+`
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Procedure  *string  `json:"procedure"`
			Since      *string  `json:"since"`
			Between    []string `json:"between"`
			Subject    *string  `json:"subject"`
			Target     *string  `json:"target"`
			RecordKind *string  `json:"record_kind"`
			Limit      *int     `json:"limit"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		// The positional is read first and written last, which is where
		// the command takes it. It is refused empty for the reason every
		// other argument here is: `changes` reads no positional at all
		// as *compare across every Procedure*, so `procedure: ""` would
		// answer a fold over the whole Store to a caller who named one
		// (namedValue.flagged).
		procedure := ""
		if named.Procedure != nil {
			if err := namesSomething("procedure", *named.Procedure, "Procedure"); err != nil {
				return nil, err
			}
			procedure = *named.Procedure
		}

		since, err := flagsFor(namedValue{"since", named.Since, "instant", "--since"})
		if err != nil {
			return nil, err
		}
		argv := append([]string{"changes"}, since...)
		if named.Between != nil {
			// A window has two ends, and the schema's minItems and
			// maxItems are made true here for readArguments' own
			// reason: a schema is a claim a client may or may not
			// check. One id is one end of a window, which is a
			// thing the flag itself refuses in as many words
			// (flags.go).
			if len(named.Between) != 2 {
				return nil, fmt.Errorf("between takes two Run ids, the baseline and the subject, and was given %d: a window has two ends", len(named.Between))
			}
			for i, id := range named.Between {
				if err := namesSomething(fmt.Sprintf("between[%d]", i), id, "Run"); err != nil {
					return nil, err
				}
			}
			argv = append(argv, "--between", named.Between[0], named.Between[1])
		}
		subject, err := flagsFor(namedValue{"subject", named.Subject, "Run", "--subject"})
		if err != nil {
			return nil, err
		}
		argv = append(argv, subject...)
		narrowed, err := flagsFor(
			namedValue{"target", named.Target, "Target", "--target"},
			namedValue{"record_kind", named.RecordKind, "Record type", "--kind"},
		)
		if err != nil {
			return nil, err
		}
		argv = append(argv, narrowed...)
		argv = append(argv, cappedAt(named.Limit)...)
		if procedure == "" {
			return argv, nil
		}
		// The positional last and past one `--`, which is the command
		// line the command would have received: a Procedure name is
		// matched byte-exact against a namespace that can hold a name
		// spelled like a flag (flags.go).
		return append(argv, "--", procedure), nil
	},
}

// windowSide is one end of a Comparison's window as a schema, written once
// because a `window` row carries two of them and they are one shape (§8,
// compare.SideRow).
//
// **`ended` stands where the page renders a duration**: §7 is precise that no
// duration is stored anywhere, and the wire carries the two instants it
// subtracted and never the subtraction. Its absence is what the page renders
// `reaped` for, and `closed_by` beside it says why.
const windowSide = `{
	"type": "object",
	"additionalProperties": false,
	"required": ["run", "trigger", "started", "dry_run", "outcome", "procedure_revision"],
	"properties": {
		"run": {"type": "string"},
		"trigger": {"type": "string"},
		"started": {"type": "string"},
		"dry_run": {
			"type": "boolean",
			"description": "Whether that Run was a rehearsal. Written always, the bare false included — §7's one exception to the absence rule, on the surface that names two Runs. A baseline is never a rehearsal; a subject is one where the caller named it."
		},
		"outcome": {
			"enum": ["completed", "refused", "failed"],
			"description": "The entry's own, written always: a window never names an open entry, so there is always one."
		},
		"ended": {"type": "string", "description": "Absent on a reaped entry, whose only account is a closing write on the closing Run's clock, so no duration derives."},
		"procedure_revision": {"type": "string"},
		"repo_dirty": {"type": "boolean"},
		"closed_by": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["run", "outcome", "ended"],
				"properties": {
					"run": {"type": "string"},
					"outcome": {"const": "failed"},
					"step": {"type": "integer", "minimum": 1},
					"ended": {"type": "string"}
				}
			}
		}
	}
}`

// recordsTool carries `hyper records` — the surface whose job is finding a
// version. `changes` reads a change and this finds the version that change is
// of.
//
// Its arguments are the identity's three columns, the boolean that opens a
// series, the window that bounds one, and the cap. **`since` is legal only with
// `history`, exactly as the flags are**: without it the parameter would filter
// Heads by when they last moved, which is a change read on the command whose
// job is finding a version — and having it turn `history` on instead would be
// the mode ADR-0013 refused. The pair the command refuses together is refused
// together here, arriving as a JSON-RPC error, being a malformed call rather
// than a guardrail declining (§9, ADR-0060, records.go).
//
// **`limit` counts identities and never rows.** Under `history` a series comes
// back whole or does not come back, a series cut partway through being a
// partial history wearing a complete one's shape; what bounds one series is a
// constant this implementation picks, and a marker naming the time axis is what
// says it cut (records.go).
//
// **`dry_run` rides on the row because the join was costing a call.** A
// rehearsal performs the reads it reaches and records Observations like any
// other Run (§6), so the versions holding a pre-state can all be a rehearsal's
// — and §7 tells every consumer of Journal evidence to filter a rehearsal out,
// which is the opposite rule pointed at the same Run. An agent reading a row
// and reaching for `run_show` to learn which rule applies is the call this
// member removes (ADR-0114, issue #234).
var recordsTool = tool{
	name:        "records",
	text:        locationBeneathSummary,
	description: "Find a version: one row per Record with its ordinal, the Run and Step that wrote it, whether that Run was a rehearsal, its state and its Provenance — or every version of one, under history.",
	input: closedObject(`{
		"target": {"type": "string", "minLength": 1, "description": "The identity's first column, matched byte-exact over UTF-8."},
		"definition": {"type": "string", "minLength": 1, "description": "Its second."},
		"name": {"type": "string", "minLength": 1, "description": "Its third. Naming all three is naming one Record, and naming none is the whole branch."},
		"history": {
			"type": "boolean",
			"description": "Whether to answer every version of each Record rather than its Head alone, newest first. It is an explicit boolean and never a mode another argument turns on."
		},
		"since": {
			"type": "string",
			"description": "An RFC 3339 instant bounding the versions inside each series, inclusive of the instant it names. It is legal only with history: a Head has no window to bound, and naming one here would not open a history."
		},
		"limit": {"type": "integer", "minimum": 1, "description": "The cap, counting Records and never rows: under history a series comes back whole or does not come back."}
	}`),
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "key", "ordinal", "run_id", "step", "record_kind", "provenance"],
				"properties": {
					"type": {"const": "record"},
					"key": {
						"type": "object",
						"additionalProperties": false,
						"required": ["target", "definition", "name"],
						"description": "The Record's identity as the one fact it is: a Record is identified by its Target, its Definition and a name together, and three siblings of ordinal would be three arguments that happen to be adjacent.",
						"properties": {
							"target": {"type": "string"},
							"definition": {"type": "string"},
							"name": {"type": "string"}
						}
					},
					"ordinal": {
						"type": "integer",
						"minimum": 1,
						"description": "The version's position in the series' own ordering, stored nowhere and never its identifier — which is the Run that wrote it. It is unstable under Compaction, which is affordable for exactly one reason: nothing anywhere accepts an ordinal as input."
					},
					"run_id": {"type": "string", "description": "Whole. The Run and the Step together are the version's identity: two Steps of one Run writing one identity write two paths, so the Run alone would not name one."},
					"step": {"type": "integer", "minimum": 1},
					"dry_run": {
						"type": "boolean",
						"description": "Whether the Run that wrote this version was a rehearsal, read off that Run's Journal entry. It is written always, the bare false included: a rehearsal performs the reads it reaches and records Observations like any other Run, so a version worth reading can come from a Run every consumer of Journal evidence filters out, and a reader that takes absence for false gets that backwards. Absent only where the branch holds no entry for the Run at all."
					},
					"record_kind": {"enum": ["asset", "observation"]},
					"tombstoned": {"type": "boolean", "description": "Whether the Record's Head is a Tombstone. It is the series' state rather than the version's, so it means one thing on every row of a history."},
					"orphaned": {"type": "boolean", "description": "An Asset still standing whose Definition no longer exists. Reported for as long as it stands rather than once, or a forgotten resource becomes invisible by way of a tidy-up commit."},
					"secret_fields": {
						"type": "array",
						"items": {"type": "string"},
						"description": "The fields whose value the presence-only marker stands in for: the names, and never a value. Absent where nothing was suppressed."
					},
					"provenance": {
						"type": "object",
						"additionalProperties": false,
						"description": "The whole of it, the Run-wide half and the Step's half under one key, which is what a Record version carries and what no Journal file does. It is written always: a version states which code performed the Run that wrote it.",
						"properties": {`+provenanceMembers+`
						}
					}
				}
			}
		},
		"truncated": `+cutOrComplete+`
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		var named struct {
			Target     *string `json:"target"`
			Definition *string `json:"definition"`
			Name       *string `json:"name"`
			History    bool    `json:"history"`
			Since      *string `json:"since"`
			Limit      *int    `json:"limit"`
		}
		if err := readArguments(arguments, &named); err != nil {
			return nil, err
		}
		narrowed, err := flagsFor(
			namedValue{"target", named.Target, "Target", "--target"},
			namedValue{"definition", named.Definition, "Definition", "--definition"},
			namedValue{"name", named.Name, "Record", "--name"},
		)
		if err != nil {
			return nil, err
		}
		argv := append([]string{"records"}, narrowed...)
		if named.History {
			argv = append(argv, "--history")
		}
		// `since` is passed whether or not `history` was given, and the
		// pair is refused by the command rather than here: what the two
		// together mean is the command's own rule, and a tool that
		// decided it would be a second reading of a fact the CLI half
		// already states in a sentence a caller reads (records.go).
		since, err := flagsFor(namedValue{"since", named.Since, "instant", "--since"})
		if err != nil {
			return nil, err
		}
		argv = append(argv, since...)
		return append(argv, cappedAt(named.Limit)...), nil
	},
}

// projectTool carries `hyper project` — §9's one Lifecycle tool, and the one
// tool on this surface that **writes a file into the working tree** (issue
// #203).
//
// **It is why the line between what an agent may reach and what it may not
// falls where it does** rather than around writing at all. `install`, `store
// init` and `compact` are the three commands §9 puts on the far side; `project`
// is on the reachable side because a Cadence declared in a reviewed artefact
// and left unprojected is the drift §10 states a check for, and an agent must
// be able to repair what it caused. What it writes is derived from artefacts
// already reviewed and lands in a diff like everything else (§9, §10).
//
// **It takes no arguments at all.** The command is repo-wide and all-or-nothing
// — there is no `project <procedure>`, since per-Procedure projection would let
// two Procedures pin different versions against one Store — so there is nothing
// here for a per-Procedure argument to name, and the closed empty object is
// what refuses one (§9, ADR-0060, project.go).
//
// **It carries no `outcome` key**, and the table is where that is declared: it
// is not a Run, so it leaves `executes` nil and the envelope carries none (§9,
// envelope.go). What ends its rows is `result`, so `truncated` is the bare
// `false` on every ordinary return — `project` names no namespace to range
// over, its command carries no `--limit`, and a projection cut at a cap would
// be one nobody could review against the artefacts. It is `null` where the
// command opened no row stream at all, which here is the Refusal below.
//
// **Everything it does on the way is the command's, unchanged**: the whole-file
// overwrite of each workflow, the two scalars edited in the Repository
// declaration with every other byte carried through, the version pin derived
// from the binary that ran it, the `AGENTS.md` created where the repository
// holds none and never taken where one stands, and the
// `release-artefact-absent` Refusal where it cannot resolve a published
// artefact for its own version — which arrives here as `isError: true` with the
// rendering whole, like any other guardrail declining, and with nothing written
// (§11, ADR-0020, ADR-0095).
//
// **The description names the note even though no row reports it**, on the same
// ground the declaration is named there: a caller is told what a call writes
// into its tree, and a third file appearing in a diff that the tool's own
// description did not mention is a surprise this surface has no way to explain
// (§9, issue #211).
//
// **And it stands outside the pin gate on this surface exactly as it does on
// the CLI**, being the pin's only writer: a writer gated on what it writes is a
// bootstrap with no bootstrap. The exemption is the command's own and this
// inherits it by going through the same dispatch, which is the point of a tool
// being a schema and an argv (§9, §11, RunProject).
var projectTool = tool{
	name:        "project",
	description: "Regenerate the projection: one .github/workflows/ file per Procedure that declares a Cadence, and the two derived scalars in the Repository declaration. It is repo-wide and all-or-nothing — those files are overwritten whole, and the ones no Procedure asks for any more are taken away. It also writes an AGENTS.md carrying this orientation where the repository holds none, and never touches one that already stands. It reports the workflows it wrote, the git diff being where a human reviews all of it.",
	input:       noArguments,
	output: closedObject(`{
		"rows": {
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": false,
				"required": ["type", "path"],
				"description": "One file this call wrote or took away: one row per Procedure declaring a Cadence, all of them, and one per generated file no Procedure asks for any more. Rows are ordered by the Procedure name the path itself carries, which is the only name a removed file naming no Procedure has. It carries no last Journal entry — project writes a file and reports what it wrote, and what stands in the Store is no part of that.",
				"properties": {
					"type": {"const": "workflow"},
					"path": {"type": "string", "description": "The file, relative to the repository root, with forward slashes on every platform."},
					"procedure": {"type": "string", "description": "The Procedure that asked for the file. It is absent on a removed file whose name matches no Procedure the repository holds, and a removed file carries no gloss beside it either: the absence is the fact rather than a widening of the shape."},`+cadenceGlossMembers+`
				}
			}
		},
		"truncated": {"type": "boolean"}
	}`, "rows", "truncated"),
	argv: func(arguments json.RawMessage) ([]string, error) {
		if err := readArguments(arguments, &struct{}{}); err != nil {
			return nil, err
		}
		return []string{"project"}, nil
	},
}

// namedValue is one optional argument that carries a value: what it is called
// here, what the caller wrote, the noun a refusal names, and the flag its
// command takes.
//
// The four Inspection tools take eleven of these between them and every one of
// them reads the same way — absent is no narrowing, empty is a malformed call,
// and present is a flag and its value — so it is stated once. A tool that
// spelled the reading per argument would be eleven chances for one of them to
// let the empty string through. The two arguments that are not one of these are
// the two that do not become a flag and a value: `changes`'s positional, and the
// pair `between` takes.
type namedValue struct {
	argument string
	// value is what the caller wrote, and it is a pointer because
	// **absent and empty are two different calls**: every one of these
	// arguments narrows something, so a tool that read the two as one
	// would answer *everything* to a caller who asked about a name that
	// happens to be the empty string (flagged below).
	value *string
	noun  string
	flag  string
}

// flagged is the argument as a command line carries it: the flag and its value
// where the caller named one, nothing at all where they did not, and an error
// where they named the empty string.
//
// **The empty string is a malformed call and not an absent parameter**, which
// is namesSomething's reading and is load-bearing on every one of these: the
// command reads an empty `--target` as no narrowing at all, so a caller asking
// about a Target named `""` would be handed the whole Journal as though they
// had asked for it (check's own `paths: [""]`, one file up).
//
// The value goes after the flag as a separate argument rather than joined with
// `=`, which is the spelling every one of these flags takes and the one that
// needs no escaping: a value beginning `--` is the next argument, and the
// command's own reader takes whatever stands there (flags.go).
func (v namedValue) flagged() ([]string, error) {
	if v.value == nil {
		return nil, nil
	}
	if err := namesSomething(v.argument, *v.value, v.noun); err != nil {
		return nil, err
	}
	return []string{v.flag, *v.value}, nil
}

// flagsFor is a run of optional arguments as the command line carries them, in
// the order they were given, and the first refusal among them.
//
// It is here rather than a loop at each of the four tools because what the loop
// holds is the reading rather than the order: an argument that will not stand is
// a malformed call, and five copies of *check the error before appending* are
// five chances for one of them to append and carry on. What each tool still
// states for itself is which arguments it takes and in which order, that being
// §9's signature and not this function's.
func flagsFor(parameters ...namedValue) ([]string, error) {
	var argv []string
	for _, parameter := range parameters {
		flagged, err := parameter.flagged()
		if err != nil {
			return nil, err
		}
		argv = append(argv, flagged...)
	}
	return argv, nil
}

// cappedAt is the `--limit` a caller named, and nothing where they named none —
// which is the command's own default applying, exactly as it does on a listing
// tool that offers no cap at all (providersTool).
//
// It takes a pointer because **zero is a value this argument must be able to
// carry**: the schema's `minimum` refuses it, and a tool that read an absent
// argument and an explicit `0` as one value would silently answer the default
// where a caller asked for a cap the command refuses in its own words — a limit
// of none is the flag left off, and a limit of zero is a question with no
// answer in it (flags.go).
func cappedAt(limit *int) []string {
	if limit == nil {
		return nil
	}
	return []string{"--limit", strconv.Itoa(*limit)}
}
