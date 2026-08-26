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
// §9 states thirteen tools, each named for the command it carries. Four are
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
// §9's table states, then the repository.
var tools = []tool{providersTool, providerTool, operationTool, targetsTool}

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

// namesSomething is the empty-name reading every tool that takes a name shares,
// and the whole of what it holds is that **the server is where a schema's claim
// is made true**: `minLength` is a claim a client may or may not check, and a
// name that is well-typed and names nothing is a malformed call (§9), which is
// what an error here becomes.
//
// It takes the argument's name and the noun apart, because the two are not one
// word: `operation(provider, operation)` names two namespaces and each message
// has to say which of them was asked. The sentence is composed once so that
// three arguments cannot come to decline in three voices.
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
