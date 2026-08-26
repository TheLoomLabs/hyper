package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The tool set (§9, issue #195).
//
// **A tool is a schema, an argv, and nothing else.** Each declares its
// arguments typed and closed exactly as the flag or the positional it carries
// is, builds the command line its command would have received, and hands it to
// the same dispatch behind the server's destination. §9 fixes that *ergonomics
// is the whole of the difference between the two*; a tool that reached past the
// argv would be a second place for a guardrail to be skipped, a Refusal to be
// reworded or a row to be reshaped.
//
// §9 states thirteen tools, each named for the command it carries. Two are
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
// §9's table states.
var tools = []tool{providersTool, providerTool}

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
		// The empty name is refused here as well as in the schema, on
		// readArguments' own reading: a schema is a claim a client may or
		// may not check, and the server is where the claim is made true. A
		// name that is well-typed and names nothing is a malformed call
		// (§9), which is what an error here becomes.
		if named.Name == "" {
			return nil, errors.New("name is the empty string, which names no Provider")
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
