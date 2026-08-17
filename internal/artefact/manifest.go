// This file is the Manifest's own schema and the checks that read it against
// itself: kind: against providers/, provider: against the file's basename,
// the request written under exactly one Capability, the input-schema
// subset, and the path and template-hole grammars (§3, §4, §12, issue #91).
// It does not check a Manifest's declarations against each other —
// capability-mismatch, manifest-inconsistent and the rest of §4's "Manifest's
// oracle" cross-check what an Operation's own request and input schema
// imply against what the Manifest declares elsewhere, which is #92's.
//
// opaque is not among the keys any schema here admits — it is never a
// writable key at all, being a property of the Capability an Operation's
// request uses rather than a fact the Operation states (§3) — so a Manifest
// author who writes it earns unknown-key like any other key the schema at
// that position does not define.
package artefact

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// CodeSchemaUnsupported is the code an input schema reaching outside the
// four-keyword subset earns — type, enum, properties, items, and nothing
// else (§4, §12).
const CodeSchemaUnsupported = "schema-unsupported"

// CodeHoleIllegal is the code a template hole earns wherever it resolves
// outside its position's legal source, or stands in a position §12 does not
// list at all — inside an Auth scheme's parameters, the one position with
// no legal source at all, and a body: mapping key, which is no position at
// all (§3, §4, §12).
const CodeHoleIllegal = "hole-illegal"

// KindProvider is the one kind: value a file in providers/ may carry, and
// the one the built-in shell Provider authors outright, having no file
// (§12's kind table).
const KindProvider = "provider"

// ManifestDeclaration is a Manifest's own top-level schema (§3): its name,
// an explicit schema-version — the one artefact that carries one, the
// repository-wide version pin not reaching its author — the class: of
// Target its Definitions may bind, the capabilities: it requires, the
// auth: scheme it authenticates with if it authenticates at all, any
// enumerations: its Capability-relevant holes draw on, operations: keyed by
// Operation name, and, on an installed Manifest, the origin: block hyper
// itself writes. additionalProperties: false is forced rather than
// authored (§12), so a sixth key is unknown-key wherever it appears.
var ManifestDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "kind", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "provider", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "schema-version", Required: true, Schema: schema.Schema{Type: schema.Integer}},
		{Name: "class", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "capabilities", Required: true, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String, Enum: []string{"http", "shell"}},
		}},
		// auth, enumerations and operations are each mappings keyed by a
		// name the repository or the Provider author chooses rather than a
		// fixed set hyper enumerates, so the generic engine stops at "is
		// this a mapping" and checkAuth, checkEnumerations and
		// checkOperations below read what is inside them, the way
		// checkCredentialSlots already does for a Target declaration's
		// auth: (§4).
		{Name: "auth", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "enumerations", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "operations", Required: true, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "origin", Required: false, Schema: schema.Schema{
			Type: schema.Object,
			Properties: []schema.Property{
				{Name: "ref", Required: true, Schema: schema.Schema{Type: schema.String}},
				{Name: "digest", Required: true, Schema: schema.Schema{Type: schema.String}},
			},
		}},
	},
}

// operationDeclaration is an Operation's own flat schema (§3): the facts
// hyper would otherwise have to guess at, none of them nested under a
// grouping key. repeatability: admits only the two values an Operation may
// spell — run-once has none — and deadline: is mandatory, its absence
// schema-mismatch like any other missing required key.
var operationDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "kind", Required: true, Schema: schema.Schema{Type: schema.String, Enum: []string{"read", "mutate", "destroy"}}},
		{Name: "repeatability", Required: false, Schema: schema.Schema{Type: schema.String, Enum: []string{"repeatable", "skip-if-recorded"}}},
		{Name: "deadline", Required: true, Schema: schema.Schema{Type: schema.Duration}},
		{Name: "concurrency", Required: false, Schema: schema.Schema{Type: schema.Integer}},
		{Name: "patterns", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "input", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "http", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		// shell: carries no keys at all (§3): a fixed, empty Properties
		// list and Open left false is what makes any key inside it
		// unknown-key with no further check needed.
		{Name: "shell", Required: false, Schema: schema.Schema{Type: schema.Object}},
		{Name: "record", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "secret", Required: false, Schema: schema.Schema{
			Type: schema.Array, Items: &schema.Schema{Type: schema.String},
		}},
	},
}

// httpRequestDeclaration is an http: block's schema (§3). query: and
// headers: are mappings of name to string, always, so they are Open here
// and checkStringMapping below reads their values; body: carries no schema
// at all, so it is Open too and checkBody walks it on its own rule.
var httpRequestDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "method", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "host", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "path", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "query", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "headers", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "body", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "host-input", Required: false, Schema: schema.Schema{Type: schema.String}},
	},
}

// recordDeclaration is a record: block's schema (§3). identity: is a string
// rather than typed further because it is either a response path or a
// template hole, told apart and checked by checkIdentity; fields: is Open,
// a mapping of recorded name to response path.
var recordDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "identity", Required: false, Schema: schema.Schema{Type: schema.String}},
		{Name: "fields", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "over", Required: false, Schema: schema.Schema{Type: schema.String}},
	},
}

// authHeaderDeclaration and authBasicDeclaration are the two Auth schemes'
// parameter schemas, the set closed at two (§3, §12). basic: takes none,
// written {} rather than a bare scalar.
var (
	authHeaderDeclaration = schema.Schema{
		Type: schema.Object,
		Properties: []schema.Property{
			{Name: "name", Required: true, Schema: schema.Schema{Type: schema.String}},
			{Name: "prefix", Required: false, Schema: schema.Schema{Type: schema.String}},
		},
	}
	authBasicDeclaration = schema.Schema{Type: schema.Object}
)

// patternsDeclaration is the fixed three-member set a Pattern may name
// (§3, §12); each member's own shape is checked by checkPagination,
// checkPolling and the inline retry: schema below.
var patternsDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "pagination", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "polling", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "retry", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
	},
}

var (
	cursorDeclaration = schema.Schema{
		Type: schema.Object,
		Properties: []schema.Property{
			{Name: "from", Required: true, Schema: schema.Schema{Type: schema.String}},
			{Name: "into", Required: true, Schema: schema.Schema{Type: schema.Object, Open: true}},
		},
	}
	pageDeclaration = schema.Schema{
		Type: schema.Object,
		Properties: []schema.Property{
			{Name: "from", Required: true, Schema: schema.Schema{Type: schema.Integer}},
			{Name: "into", Required: true, Schema: schema.Schema{Type: schema.Object, Open: true}},
		},
	}
	pollingDeclaration = schema.Schema{
		Type: schema.Object,
		Properties: []schema.Property{
			{Name: "interval", Required: true, Schema: schema.Schema{Type: schema.Duration}},
			{Name: "until", Required: true, Schema: schema.Schema{
				Type: schema.Array, Items: &schema.Schema{Type: schema.Object, Open: true},
			}},
		},
	}
	retryDeclaration = schema.Schema{
		Type: schema.Object,
		Properties: []schema.Property{
			{Name: "attempts", Required: true, Schema: schema.Schema{Type: schema.Integer}},
		},
	}
)

// predicateOperators is the closed eleven-member operator set a predicate
// carries exactly one of (§12). checkPredicate reads only the shape — a
// field: and exactly one operator key — and none of the operand-type rules
// §4 and §6 state for it, which are #92's.
var predicateOperators = []string{
	"equals", "not_equals", "in", "exists", "absent",
	"starts_with", "ends_with", "greater_than", "less_than", "older_than", "newer_than",
}

// inputSchemaTypes is the closed scalar vocabulary an input schema's type:
// may name (§12): the six common JSON Schema primitives plus the two the
// domain forces.
var inputSchemaTypes = map[string]bool{
	"string": true, "integer": true, "number": true, "boolean": true,
	"object": true, "array": true, "duration": true, "timestamp": true,
}

// holePattern matches one template hole, {name}, anywhere in a scalar's
// text — the one hole syntax, naming what fills it and nothing more
// (§3, §12).
var holePattern = regexp.MustCompile(`\{([^{}]+)\}`)

// pathPattern is the path grammar §12 closes: $, then any number of
// .member or ["member"] segments, and nothing else. Recursive descent
// (..), array indexing ([n]) and iteration ([*]) are all outside it by
// omission.
var pathPattern = regexp.MustCompile(`^\$(?:\.[A-Za-z_][A-Za-z0-9_-]*|\["[^"]*"\])*$`)

// fieldNoRootPattern is the same grammar written without its root marker —
// a polling Pattern's until: field:, which roots at the response object in
// hand rather than at a Record, a response having paths and no declared
// names (§3, §12).
var fieldNoRootPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z_][A-Za-z0-9_-]*|\["[^"]*"\])*$`)

// CheckManifest validates a providers/ file's already-parsed root against
// ManifestDeclaration and every check that reads a Manifest against itself
// (§3, §4, §12, issue #91): kind: against providers/, provider: against
// the file's basename, and checkManifestBody's grammar checks. root is nil
// where the file parsed to no document at all; the schema check still runs
// and reports every required key the file never supplied.
func CheckManifest(file string, root *yaml.Node) []problem.Problem {
	problems := checkManifestBody(file, root)
	problems = append(problems, checkKind(file, root, KindProvider)...)
	problems = append(problems, checkName(file, root, "provider")...)
	return problems
}

// checkManifestBody is CheckManifest without the two checks that need a
// file of the Manifest's own to compare against — reused by
// CheckBuiltinShellProvider, which authors its name outright and has no
// directory for kind-mismatch to compare against either (§3, §11).
func checkManifestBody(file string, root *yaml.Node) []problem.Problem {
	problems := schema.Check(root, ManifestDeclaration, file)
	problems = append(problems, checkAuth(file, root)...)
	problems = append(problems, checkEnumerations(file, root)...)
	problems = append(problems, checkOperations(file, root)...)

	// env: is reserved across every artefact, a Manifest included, even
	// though a Manifest declares no credential slot of its own to carry it
	// (§3, §4) — reusing the walk withCredentialSlots layers over a
	// Target declaration's schema check for the same reason.
	problems = dropReservedEnvKey(problems)
	problems = append(problems, findReservedEnvKeys(file, root, "", nil)...)
	return problems
}

// checkOperations reads operations: — a mapping keyed by Operation name
// rather than a fixed shape, hence Open in ManifestDeclaration — and
// validates each entry against operationDeclaration and the request,
// input-schema and projection grammars beneath it (§3, §4, §12).
func checkOperations(file string, root *yaml.Node) []problem.Problem {
	opsVal := topLevelFields(root, "operations")["operations"]
	if opsVal == nil || opsVal.Kind != yaml.MappingNode {
		return nil
	}
	enumNames := enumerationNames(root)

	var problems []problem.Problem
	for i := 0; i+1 < len(opsVal.Content); i += 2 {
		nameNode, opNode := opsVal.Content[i], opsVal.Content[i+1]
		if nameNode.Kind != yaml.ScalarNode {
			continue
		}
		problems = append(problems, checkOneOperation(file, "operations."+nameNode.Value, opNode, enumNames)...)
	}
	return problems
}

// checkOneOperation validates one operations: entry: its own flat schema,
// exactly one request block between http: and shell:, the input-schema
// subset, the three Patterns, and the record: projection's paths and holes
// (§3, §4, §12).
func checkOneOperation(file, field string, op *yaml.Node, enumNames map[string]bool) []problem.Problem {
	problems := schema.CheckAt(op, operationDeclaration, field, file)
	if op == nil || op.Kind != yaml.MappingNode {
		return problems
	}

	problems = append(problems, checkExactlyOneOf(file, field, op, []string{"http", "shell"})...)

	fields := topLevelFields(op, "http", "input", "patterns", "record")
	inputProps := inputPropertyNames(fields["input"])

	if httpVal := fields["http"]; httpVal != nil && httpVal.Kind == yaml.MappingNode {
		problems = append(problems, checkHTTPRequest(file, field+".http", httpVal, enumNames, inputProps)...)
	}
	if inputVal := fields["input"]; inputVal != nil && inputVal.Kind == yaml.MappingNode {
		problems = append(problems, checkInputSchema(file, field+".input", inputVal)...)
	}
	if patternsVal := fields["patterns"]; patternsVal != nil && patternsVal.Kind == yaml.MappingNode {
		problems = append(problems, checkPatterns(file, field+".patterns", patternsVal)...)
	}
	if recordVal := fields["record"]; recordVal != nil && recordVal.Kind == yaml.MappingNode {
		problems = append(problems, checkRecord(file, field+".record", recordVal, inputProps)...)
	}
	return problems
}

// checkExactlyOneOf reports schema-mismatch where node carries none, or
// more than one, of keys: an Operation's exactly-one request block among
// http:/shell:, an Auth scheme's exactly-one name among header:/basic:, a
// pagination Pattern's exactly-one form among cursor:/page:, and an into:
// mapping's exactly-one position among query:/header: (§3, §4).
func checkExactlyOneOf(file, field string, node *yaml.Node, keys []string) []problem.Problem {
	found := topLevelFields(node, keys...)
	present := 0
	for _, k := range keys {
		if found[k] != nil {
			present++
		}
	}
	if present == 1 {
		return nil
	}
	line, column := position(node)
	return []problem.Problem{{
		File:      file,
		Line:      line,
		Column:    column,
		Field:     field,
		ErrorCode: schema.CodeMismatch,
		Message:   fmt.Sprintf("%s: exactly one of %s must be present", field, strings.Join(keys, ", ")),
	}}
}

// checkHTTPRequest validates an http: block's own schema, then the two
// hole rules the request's positions split on: host: is Capability-relevant
// and its holes resolve only against a declared enumerations: entry or
// from-target; every other position resolves only against this Operation's
// own input (§3, §12).
func checkHTTPRequest(file, field string, node *yaml.Node, enumNames, inputProps map[string]bool) []problem.Problem {
	problems := schema.CheckAt(node, httpRequestDeclaration, field, file)
	fields := topLevelFields(node, "method", "host", "path", "query", "headers", "body")

	if methodVal := fields["method"]; methodVal != nil && methodVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkOrdinaryHoles(file, field+".method", methodVal, inputProps)...)
	}
	if hostVal := fields["host"]; hostVal != nil && hostVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkCapabilityHoles(file, field+".host", hostVal, enumNames)...)
	}
	if pathVal := fields["path"]; pathVal != nil && pathVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkOrdinaryHoles(file, field+".path", pathVal, inputProps)...)
	}
	problems = append(problems, checkStringMapping(file, field+".query", fields["query"], inputProps)...)
	problems = append(problems, checkStringMapping(file, field+".headers", fields["headers"], inputProps)...)
	problems = append(problems, checkBody(file, field+".body", fields["body"], inputProps)...)
	return problems
}

// checkStringMapping reads a query: or headers: mapping's members, which
// are always name to string (§3): a non-scalar value is schema-mismatch,
// and a scalar's holes are checked as an ordinary position's are.
func checkStringMapping(file, field string, node *yaml.Node, inputProps map[string]bool) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var problems []problem.Problem
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, val := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		childField := field + "." + key.Value
		if val.Kind != yaml.ScalarNode {
			problems = append(problems, problem.Problem{
				File: file, Line: val.Line, Column: val.Column, Field: childField,
				ErrorCode: schema.CodeMismatch,
				Message:   "query: and headers: are mappings of name to string",
			})
			continue
		}
		problems = append(problems, checkOrdinaryHoles(file, childField, val, inputProps)...)
	}
	return problems
}

// checkBody walks a body: value tree — the one position in any artefact
// hyper holds no schema for (§3). A literal scalar is left untyped by this
// check on purpose, carrying its YAML 1.2 core type onto the wire; what
// this still checks is the two rules that hold regardless of the API's own
// shape: a hole fills a value only, never a mapping key, and a hole in a
// value resolves only against this Operation's own input.
func checkBody(file, field string, node *yaml.Node, inputProps map[string]bool) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	return checkBodyValue(file, field, node, inputProps)
}

func checkBodyValue(file, field string, node *yaml.Node, inputProps map[string]bool) []problem.Problem {
	var problems []problem.Problem
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, val := node.Content[i], node.Content[i+1]
			childField := field + "." + key.Value
			if key.Kind == yaml.ScalarNode && holePattern.MatchString(key.Value) {
				problems = append(problems, problem.Problem{
					File: file, Line: key.Line, Column: key.Column, Field: childField,
					ErrorCode: CodeHoleIllegal,
					Message:   "a hole fills a value position only — a mapping key is no position at all",
				})
			}
			problems = append(problems, checkBodyValue(file, childField, val, inputProps)...)
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			problems = append(problems, checkBodyValue(file, fmt.Sprintf("%s[%d]", field, i), item, inputProps)...)
		}
	case yaml.ScalarNode:
		problems = append(problems, checkOrdinaryHoles(file, field, node, inputProps)...)
	}
	return problems
}

// checkCapabilityHoles reads every hole in a Capability-relevant scalar —
// an Operation's host: — and reports one resolving to anything but
// from-target or a declared enumerations: entry as hole-illegal (§3, §12).
func checkCapabilityHoles(file, field string, node *yaml.Node, enumNames map[string]bool) []problem.Problem {
	var problems []problem.Problem
	for _, m := range holePattern.FindAllStringSubmatch(node.Value, -1) {
		name := m[1]
		if name == "from-target" || enumNames[name] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: node.Line, Column: node.Column, Field: field,
			ErrorCode: CodeHoleIllegal,
			Message:   fmt.Sprintf("{%s} resolves to neither from-target nor a declared enumerations: entry", name),
		})
	}
	return problems
}

// checkOrdinaryHoles reads every hole in an ordinary scalar — every
// request position but host: and an Auth scheme's parameters — and reports
// one naming anything but this Operation's own input as hole-illegal
// (§3, §12).
func checkOrdinaryHoles(file, field string, node *yaml.Node, inputProps map[string]bool) []problem.Problem {
	var problems []problem.Problem
	for _, m := range holePattern.FindAllStringSubmatch(node.Value, -1) {
		name := m[1]
		if inputProps[name] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: node.Line, Column: node.Column, Field: field,
			ErrorCode: CodeHoleIllegal,
			Message:   fmt.Sprintf("{%s} names no input this Operation declares", name),
		})
	}
	return problems
}

// checkAuthHoles reports every hole found anywhere inside an Auth scheme's
// parameters as hole-illegal outright, whatever it would have resolved to —
// the one position with no legal source at all (§3, §4, §12).
func checkAuthHoles(file, field string, node *yaml.Node) []problem.Problem {
	if node == nil {
		return nil
	}
	var problems []problem.Problem
	switch node.Kind {
	case yaml.ScalarNode:
		if holePattern.MatchString(node.Value) {
			problems = append(problems, problem.Problem{
				File: file, Line: node.Line, Column: node.Column, Field: field,
				ErrorCode: CodeHoleIllegal,
				Message:   "an Auth scheme's parameters admit no hole of any kind",
			})
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, val := node.Content[i], node.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			problems = append(problems, checkAuthHoles(file, field+"."+key.Value, val)...)
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			problems = append(problems, checkAuthHoles(file, fmt.Sprintf("%s[%d]", field, i), item)...)
		}
	}
	return problems
}

// inputPropertyNames reads the top-level property names an input: schema
// declares, or none where the Operation takes none (§3) — the set every
// ordinary hole and a record: identity: hole resolve against (§12).
func inputPropertyNames(inputVal *yaml.Node) map[string]bool {
	names := map[string]bool{}
	if inputVal == nil || inputVal.Kind != yaml.MappingNode {
		return names
	}
	propsVal := topLevelFields(inputVal, "properties")["properties"]
	if propsVal == nil || propsVal.Kind != yaml.MappingNode {
		return names
	}
	for i := 0; i+1 < len(propsVal.Content); i += 2 {
		if key := propsVal.Content[i]; key.Kind == yaml.ScalarNode {
			names[key.Value] = true
		}
	}
	return names
}

// enumerationNames reads the top-level enumerations: mapping's own names —
// the set a Capability-relevant hole may resolve against beside
// from-target (§3, §12).
func enumerationNames(root *yaml.Node) map[string]bool {
	names := map[string]bool{}
	enumVal := topLevelFields(root, "enumerations")["enumerations"]
	if enumVal == nil || enumVal.Kind != yaml.MappingNode {
		return names
	}
	for i := 0; i+1 < len(enumVal.Content); i += 2 {
		if key := enumVal.Content[i]; key.Kind == yaml.ScalarNode {
			names[key.Value] = true
		}
	}
	return names
}

// checkEnumerations validates enumerations: — a mapping of name to a list
// of bare scalars (§3), and not the input-schema subset's own enum:, which
// constrains a value a caller supplies rather than declaring a Capability
// enumeration.
func checkEnumerations(file string, root *yaml.Node) []problem.Problem {
	enumVal := topLevelFields(root, "enumerations")["enumerations"]
	if enumVal == nil || enumVal.Kind != yaml.MappingNode {
		return nil
	}
	var problems []problem.Problem
	for i := 0; i+1 < len(enumVal.Content); i += 2 {
		key, val := enumVal.Content[i], enumVal.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		field := "enumerations." + key.Value
		if val.Kind != yaml.SequenceNode {
			line, column := position(val)
			problems = append(problems, problem.Problem{
				File: file, Line: line, Column: column, Field: field,
				ErrorCode: schema.CodeMismatch,
				Message:   "enumerations: is a mapping of name to a list of bare scalars",
			})
			continue
		}
		for j, item := range val.Content {
			if item.Kind != yaml.ScalarNode {
				problems = append(problems, problem.Problem{
					File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("%s[%d]", field, j),
					ErrorCode: schema.CodeMismatch,
					Message:   "expected a bare scalar at this position",
				})
			}
		}
	}
	return problems
}

// checkAuth validates auth: — a mapping carrying exactly one key, the
// scheme's name, over that scheme's parameters (§3). The set is closed at
// header: and basic: (§12), and neither scheme's parameters admit a hole
// of any kind.
func checkAuth(file string, root *yaml.Node) []problem.Problem {
	authVal := topLevelFields(root, "auth")["auth"]
	if authVal == nil || authVal.Kind != yaml.MappingNode {
		return nil
	}
	problems := checkExactlyOneOf(file, "auth", authVal, []string{"header", "basic"})
	fields := topLevelFields(authVal, "header", "basic")
	if headerVal := fields["header"]; headerVal != nil {
		problems = append(problems, schema.CheckAt(headerVal, authHeaderDeclaration, "auth.header", file)...)
		problems = append(problems, checkAuthHoles(file, "auth.header", headerVal)...)
	}
	if basicVal := fields["basic"]; basicVal != nil {
		problems = append(problems, schema.CheckAt(basicVal, authBasicDeclaration, "auth.basic", file)...)
		problems = append(problems, checkAuthHoles(file, "auth.basic", basicVal)...)
	}
	return problems
}

// checkPatterns validates patterns: against the closed three-member set —
// pagination, polling and retry — and each member's own shape (§3, §12).
func checkPatterns(file, field string, node *yaml.Node) []problem.Problem {
	problems := schema.CheckAt(node, patternsDeclaration, field, file)
	fields := topLevelFields(node, "pagination", "polling", "retry")

	if pag := fields["pagination"]; pag != nil && pag.Kind == yaml.MappingNode {
		problems = append(problems, checkPagination(file, field+".pagination", pag)...)
	}
	if poll := fields["polling"]; poll != nil && poll.Kind == yaml.MappingNode {
		problems = append(problems, checkPolling(file, field+".polling", poll)...)
	}
	if retry := fields["retry"]; retry != nil {
		problems = append(problems, schema.CheckAt(retry, retryDeclaration, field+".retry", file)...)
	}
	return problems
}

// checkPagination validates a pagination: block: exactly one of its two
// closed forms, cursor: or page:, each carrying into:, a single-key
// mapping naming query: or header: (§3, §12).
func checkPagination(file, field string, node *yaml.Node) []problem.Problem {
	problems := checkExactlyOneOf(file, field, node, []string{"cursor", "page"})
	fields := topLevelFields(node, "cursor", "page")

	if cursor := fields["cursor"]; cursor != nil {
		problems = append(problems, schema.CheckAt(cursor, cursorDeclaration, field+".cursor", file)...)
		if cursor.Kind == yaml.MappingNode {
			problems = append(problems, checkIntoExactlyOne(file, field+".cursor.into", cursor)...)
			if fromVal := topLevelFields(cursor, "from")["from"]; fromVal != nil && fromVal.Kind == yaml.ScalarNode {
				problems = append(problems, checkPathValue(file, field+".cursor.from", fromVal)...)
			}
		}
	}
	if page := fields["page"]; page != nil {
		problems = append(problems, schema.CheckAt(page, pageDeclaration, field+".page", file)...)
		if page.Kind == yaml.MappingNode {
			problems = append(problems, checkIntoExactlyOne(file, field+".page.into", page)...)
		}
	}
	return problems
}

// checkIntoExactlyOne validates into: — a single-key mapping naming query:
// or header: (§3) — under a parent carrying it, cursor: or page:.
func checkIntoExactlyOne(file, field string, parent *yaml.Node) []problem.Problem {
	intoVal := topLevelFields(parent, "into")["into"]
	if intoVal == nil || intoVal.Kind != yaml.MappingNode {
		return nil
	}
	return checkExactlyOneOf(file, field, intoVal, []string{"query", "header"})
}

// checkPolling validates a polling: block's own schema, then reads until:'s
// members as predicates rooted at the response object in hand (§3, §12).
func checkPolling(file, field string, node *yaml.Node) []problem.Problem {
	problems := schema.CheckAt(node, pollingDeclaration, field, file)
	untilVal := topLevelFields(node, "until")["until"]
	if untilVal == nil || untilVal.Kind != yaml.SequenceNode {
		return problems
	}
	for i, item := range untilVal.Content {
		problems = append(problems, checkPredicate(file, fmt.Sprintf("%s.until[%d]", field, i), item)...)
	}
	return problems
}

// checkPredicate validates one predicate's shape: a field: and exactly one
// of the closed eleven-member operator set (§12). The operand-type rules
// §4 and §6 state for each operator are #92's — this reads only the shape
// a polling Pattern's until: needs to be well-formed at all.
func checkPredicate(file, field string, node *yaml.Node) []problem.Problem {
	problems := schema.CheckAt(node, schema.Schema{Type: schema.Object, Open: true}, field, file)
	if node == nil || node.Kind != yaml.MappingNode {
		return problems
	}
	fieldNameVal := topLevelFields(node, "field")["field"]
	if fieldNameVal == nil || fieldNameVal.Kind != yaml.ScalarNode {
		line, column := position(node)
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".field",
			ErrorCode: schema.CodeMismatch,
			Message:   `the schema at this position declares "field", and this file does not supply it`,
		})
	} else {
		problems = append(problems, checkFieldPathNoRoot(file, field+".field", fieldNameVal)...)
	}
	problems = append(problems, checkExactlyOneOf(file, field, node, predicateOperators)...)
	return problems
}

// checkRecord validates a record: block's own schema, then its paths: an
// identity: that is either a response path or a template hole, an over:
// that is always a response path, and every fields: entry, a response path
// (§3, §12).
func checkRecord(file, field string, node *yaml.Node, inputProps map[string]bool) []problem.Problem {
	problems := schema.CheckAt(node, recordDeclaration, field, file)
	if node == nil || node.Kind != yaml.MappingNode {
		return problems
	}
	fields := topLevelFields(node, "identity", "fields", "over")

	if identityVal := fields["identity"]; identityVal != nil && identityVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkIdentity(file, field+".identity", identityVal, inputProps)...)
	}
	if overVal := fields["over"]; overVal != nil && overVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkPathValue(file, field+".over", overVal)...)
	}
	if fieldsVal := fields["fields"]; fieldsVal != nil && fieldsVal.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(fieldsVal.Content); i += 2 {
			key, val := fieldsVal.Content[i], fieldsVal.Content[i+1]
			if key.Kind != yaml.ScalarNode || val.Kind != yaml.ScalarNode {
				continue
			}
			problems = append(problems, checkPathValue(file, field+".fields."+key.Value, val)...)
		}
	}
	return problems
}

// checkIdentity reads a record: identity: value, which is either a
// response path or — the one position a template hole reaches into a
// projection, for a skip-if-recorded Operation whose test must resolve
// before the call — a hole naming an Operation input (§3, §12). The two
// forms are told apart by the first character, the way every scalar in
// this grammar is.
func checkIdentity(file, field string, node *yaml.Node, inputProps map[string]bool) []problem.Problem {
	switch {
	case strings.HasPrefix(node.Value, "$"):
		return checkPathValue(file, field, node)
	case strings.HasPrefix(node.Value, "{"):
		return checkOrdinaryHoles(file, field, node, inputProps)
	default:
		return []problem.Problem{{
			File: file, Line: node.Line, Column: node.Column, Field: field,
			ErrorCode: schema.CodeMismatch,
			Message:   fmt.Sprintf("%q is neither a response path nor a template hole", node.Value),
		}}
	}
}

// checkPathValue reports schema-mismatch where node's text is not a
// well-formed path in the closed grammar: $, .member and ["member"], and
// nothing else (§12).
func checkPathValue(file, field string, node *yaml.Node) []problem.Problem {
	if pathPattern.MatchString(node.Value) {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: node.Line, Column: node.Column, Field: field,
		ErrorCode: schema.CodeMismatch,
		Message:   fmt.Sprintf("%q is not a well-formed path — the grammar is $, .member and [\"member\"]", node.Value),
	}}
}

// checkFieldPathNoRoot is checkPathValue for a polling Pattern's until:
// field:, written without the root marker a response has no declared names
// to follow (§3, §12).
func checkFieldPathNoRoot(file, field string, node *yaml.Node) []problem.Problem {
	if fieldNoRootPattern.MatchString(node.Value) {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: node.Line, Column: node.Column, Field: field,
		ErrorCode: schema.CodeMismatch,
		Message:   fmt.Sprintf("%q is not a well-formed field — the grammar is $, .member and [\"member\"], written without its root marker", node.Value),
	}}
}

// checkInputSchema validates an Operation's input: against the input-schema
// subset (§12): four keywords, type, enum, properties and items, and
// nothing else — additionalProperties: false is forced by the subset's own
// closure rather than authored, so nothing here ever admits a fifth
// (§3, §4).
func checkInputSchema(file, field string, node *yaml.Node) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode {
		line, column := position(node)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: schema.CodeMismatch, Message: "an input schema is a mapping",
		}}
	}

	var problems []problem.Problem
	fields := topLevelFields(node, "type", "enum", "properties", "items")

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		switch key.Value {
		case "type", "enum", "properties", "items":
		default:
			problems = append(problems, problem.Problem{
				File: file, Line: key.Line, Column: key.Column, Field: field + "." + key.Value,
				ErrorCode: CodeSchemaUnsupported,
				Message:   fmt.Sprintf("%q reaches outside the input-schema subset — type, enum, properties and items are the whole of it", key.Value),
			})
		}
	}

	if typeVal := fields["type"]; typeVal != nil && (typeVal.Kind != yaml.ScalarNode || !inputSchemaTypes[typeVal.Value]) {
		problems = append(problems, problem.Problem{
			File: file, Line: typeVal.Line, Column: typeVal.Column, Field: field + ".type",
			ErrorCode: schema.CodeMismatch,
			Message:   "expected one of the closed scalar vocabulary at this position",
		})
	}
	if enumVal := fields["enum"]; enumVal != nil && enumVal.Kind != yaml.SequenceNode {
		problems = append(problems, problem.Problem{
			File: file, Line: enumVal.Line, Column: enumVal.Column, Field: field + ".enum",
			ErrorCode: schema.CodeMismatch, Message: "enum: is a list of bare scalars",
		})
	}
	if propsVal := fields["properties"]; propsVal != nil {
		if propsVal.Kind != yaml.MappingNode {
			problems = append(problems, problem.Problem{
				File: file, Line: propsVal.Line, Column: propsVal.Column, Field: field + ".properties",
				ErrorCode: schema.CodeMismatch, Message: "properties: is a mapping of name to schema",
			})
		} else {
			for i := 0; i+1 < len(propsVal.Content); i += 2 {
				key, val := propsVal.Content[i], propsVal.Content[i+1]
				if key.Kind != yaml.ScalarNode {
					continue
				}
				problems = append(problems, checkInputSchema(file, field+".properties."+key.Value, val)...)
			}
		}
	}
	if itemsVal := fields["items"]; itemsVal != nil {
		problems = append(problems, checkInputSchema(file, field+".items", itemsVal)...)
	}
	return problems
}
