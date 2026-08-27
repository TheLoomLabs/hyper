package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The tool set (§9, issues #195 and #197).
//
// **A tool is a schema, an argv, and nothing else.** Each declares its
// arguments typed and closed exactly as the flag or the positional it carries
// is, builds the command line its command would have received, and hands it to
// the same dispatch behind the server's destination. §9 fixes that *ergonomics
// is the whole of the difference between the two*; a tool that reached past the
// argv would be a second place for a guardrail to be skipped, a Refusal to be
// reworded or a row to be reshaped.
//
// §9 states thirteen tools, each named for the command it carries. Six are
// here. The rest arrive with the milestones that build them, on tree.go's own
// rule for the command surface: a name is real when the code behind it is, and
// the table is where that becomes true rather than a list to be kept in step.

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
	// rendersInFull is §9's text-block table read as a property of the tool,
	// which is how §9 states it: *any ordinary return* carries one summary
	// line and **`review`** carries the full rendered review surface. The
	// third row of that table is a Refusal's, which is a property of the
	// path rather than of the tool and is therefore not here.
	//
	// It is a bit on the tool rather than a composition the tool supplies,
	// because what it selects between is two readings of one answer the
	// destination already retained — the rows counted, or the page the
	// command wrote (destination.go). A tool that composed its own text
	// would be a tool holding a rendering, which is the one thing *a tool
	// is a schema, an argv, and nothing else* forbids.
	rendersInFull bool
}

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
// §9's table states, then the repository, then Authoring.
var tools = []tool{providersTool, providerTool, operationTool, targetsTool, checkTool, reviewTool}

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
								"present": {
									"type": "boolean",
									"description": "Whether that variable is set, computed when the tool runs. It is absent, with env, on a slot naming no variable: there is nothing to ask the environment about, and false would answer a question nothing asked."
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

// checkTool carries `hyper check [path...]` — the first of §9's two Authoring
// tools, and the first tool on this surface that answers **pass or fail**
// rather than a fact about the repository.
//
// It is positioned so that the next act is an edit: a row is a file, a line, a
// column, the field, the `error_code` §12 names the check by, and a message,
// which is what makes *report and then edit* practical for an agent for the
// same reason it is for a human (§9, ADR-0001).
//
// **`paths` is the CLI's positional list arriving as one typed argument**, and
// it narrows what is *reported* and never what is *loaded*: every rule §4
// states compares one artefact against another, so a subset of a repository is
// not checkable on its own — only reportable on its own (§9, check.go). The
// argument is optional because the command's positional list is, and the
// absent list is every problem the repository has rather than none.
//
// **The paths resolve where the command resolves them, which is against the
// process's working directory and not against the repository root**, and §9's
// sketch calls them *repository-relative* — a disagreement this tool inherits
// rather than settles. It cannot settle it: a tool builds the command line its
// command would have received and holds no logic of its own, and it has no
// repository in hand to re-root a path against even if it did. Nor is the
// command obviously wrong — a person typing `hyper check a.yaml` from inside
// `definitions/` means the file beside them, and repository-relative would
// refuse it. What this surface can do is state which root it is, which the
// schema above does, and leave the two spellings of one argument to a ticket
// of their own (§9, check.go, issue #198).
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
	input: closedObject(`{
		"paths": {
			"type": "array",
			"items": {"type": "string", "minLength": 1},
			"description": "Paths as the command's own positionals take them: relative to the working directory this server was started in, or absolute. Every artefact still loads and only the problems positioned in the ones named are reported; omit it to report every problem the repository has."
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
		// A member is refused for the reason a name is, and the reading is
		// load-bearing rather than tidy here: the command resolves a path
		// against the working directory before it stats one, so the empty
		// string resolves to that directory, stats clean as a directory
		// does, and then matches no problem's file — `check([""])` would
		// answer *no problems found* over a repository full of them. The
		// index is in the message because the list has more than one place
		// to be wrong in.
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
// **the one tool §9's text-block table names**: its `text` is the whole rendered
// review surface, the gutter and `AUTHORITY` and `FLAGS` exactly as the command
// writes them to stdout.
//
// That is the point of the tool rather than a convenience of it. An agent can
// read what a human reviewer will read and hand it to them verbatim, before
// asking them to read it — which is the same trade §8 makes for a Refusal, and
// it is made for the same reason: with no bypass anywhere the rendering is the
// whole of what a reviewer is given (§9, ADR-0001, ADR-0026).
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
	input: closedObject(`{
		"artefact": {
			"type": "string",
			"minLength": 1,
			"description": "The artefact to review: a repository-relative path — one containing / or ending .yaml — or the name the artefact declares for itself."
		}
	}`, "artefact"),
	rendersInFull: true,
	output: closedObject(`{
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
							"cadence": {"type": "string", "description": "The recurrence expression exactly as the artefact wrote it."},
							"phrase": {"type": "string"},
							"rate": {"type": "number", "description": "Runs per month at the two significant figures §10 rounds to. Zero is a rate: an expression the calendar has no instance of matches nothing."},
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
	}`, "rows", "truncated"),
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
// **`additionalProperties` is false on every schema this surface publishes.**
// §9's arguments are closed sets, and a schema that admitted a member it does
// not state would be one under which an override argument is well-formed —
// which is the one thing *no tool takes an override argument of any kind, under
// any name* has to be held by. The same closure on the output is what makes the
// envelope's structured half a contract rather than a description: a member the
// schema does not name is a member this surface does not write.
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
