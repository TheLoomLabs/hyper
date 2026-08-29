// This file is the Manifest's own schema, the checks that read it against
// itself — kind: against providers/, provider: against the file's basename,
// the request written under exactly one Capability, the input-schema
// subset, and the path and template-hole grammars (§3, §4, §12, issue #91)
// — and the Manifest's oracle: the checks that read a Manifest's own
// declarations against each other, with nothing but the file in hand
// (§4, issue #92). capability-mismatch and identity-undeclared read what an
// Operation's own request and record: imply against what the Manifest
// declares elsewhere; manifest-inconsistent is twelve decidable-from-one-
// file shapes of one fact sharing one code; header-reserved and
// capability-reserved refuse a name the tool holds rather than an internal
// contradiction, the second of them being the one check here whose subject
// is not the Manifest but where it was loaded from — §11's rule that an
// Extension may never hold the shell Capability, which is why it runs in
// CheckManifest and not in the body the built-in shares (issue #186). One
// further shape of manifest-inconsistent — Target slot coverage — needs a
// (Definition, Target) binding neither this file nor this milestone's
// artefacts supplied on their own, and is definition.go's, alongside
// target-class-mismatch, a code of its own that needs the same binding
// (issue #93).
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

// CodeCapabilityMismatch is the code a Manifest's declared capabilities:
// earns for disagreeing with what hyper derives from every Operation's own
// request block — over-declared or under, either direction (§3, §4).
const CodeCapabilityMismatch = "capability-mismatch"

// CodeCapabilityReserved is the code a Manifest loaded from providers/ earns
// for reaching a Capability reserved to the Providers hyper ships — shell,
// the one behind an opaque Operation, so that *a third party can never ship
// a Provider that runs commands on your machine*, which is the honest form
// §13 states the guarantee in (§11, §12, ADR-0004).
//
// It is drawn on a name the tool holds rather than on an internal
// contradiction, which is header-reserved's shape one artefact-class over
// (§4): what capability-mismatch reads is whether a Manifest agrees with
// itself, and a Manifest can reach this one while agreeing with itself
// exactly.
const CodeCapabilityReserved = "capability-reserved"

// CodeIdentityUndeclared is the code an Operation projecting a Record and
// declaring no identity: for it earns — a Record's name is the value the
// identity field holds, and a Record with none produces one nothing can
// identify (§3, §4).
const CodeIdentityUndeclared = "identity-undeclared"

// CodeManifestInconsistent is the one code twelve decidable-from-one-
// Manifest shapes of a Manifest disagreeing with itself share, each
// pointing a reader at one file, one Operation, and two adjacent keys
// rather than earning a code of its own (§3, §4). The thirteenth shape —
// Target slot coverage — needs a (Definition, Target) binding to decide
// and is #93's, and the fourteenth — a candidate set and a bound Target's
// grant intersecting to several hosts under an Operation declaring no
// host-input: — needs a Step's binding and is #98's, emitted from
// procedure.go where that binding is read.
//
// The twelfth of the twelve is a path: carrying a ? or a #. The ? is the
// two adjacent keys in the plainest form the code has — the value is in
// path: and it belongs in query: — and the # is the same fault with no key
// to move to, a fragment being a thing no request carries at all
// (ADR-0107, issue #229).
const CodeManifestInconsistent = "manifest-inconsistent"

// CodeHeaderReserved is the code drawn on the five headers hyper computes
// for itself — Host, Content-Length, Content-Type, Transfer-Encoding,
// Connection — earns wherever a second writer names one, compared
// case-insensitively: an Auth scheme's name: parameter and an ordinary
// headers: entry alike, one check with two writers (§3, §4, §12).
const CodeHeaderReserved = "header-reserved"

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
// carries exactly one of (§12). checkPredicateCore, in procedure.go, reads
// both this shape and each present operator's own operand-type rule; the
// operand-type rules themselves are #97's.
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

// reservedHeaders is the five headers hyper computes for itself, reserved
// against every writer — an Auth scheme's name: parameter and an ordinary
// headers: entry alike, compared case-insensitively as an HTTP header name
// is (§4, §12).
var reservedHeaders = map[string]bool{
	"host": true, "content-length": true, "content-type": true,
	"transfer-encoding": true, "connection": true,
}

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
//
// It is also where §11's two rules about a Manifest **loaded from providers/**
// run, and the placement is the criterion: what capability-reserved and
// origin-digest-mismatch are about is where the file came from, which is what
// calling this function already means — artefactChecks routes the built-in to
// CheckBuiltinShellProvider and every providers/ file here (issues #186, #189).
//
// manifest is the exact bytes root parsed from, which the load keeps beside
// every artefact for manifest_digest's reason and for `operation`'s (§7, §9,
// internal/repository). It stands where every sibling check's extra arguments
// stand, after the artefact's own two: it is what this check needs **beyond**
// the parse tree rather than a second spelling of it — the digest an installed
// Manifest records covers a byte range of the file, which is not a thing a
// parse tree holds (§11, manifest_origin.go).
//
// **One check stands ahead of all of them and replaces them.** A Manifest
// declaring a schema version above the one this binary reads is one code and
// nothing else, this reader having no claim on the shape of the keys beneath it
// (§11, ADR-0028, manifest_schema.go). It is written as a return rather than as
// a suppression each check consults, which is the same argument the drop in
// withReservedCapability makes one row at a time: a check that has to know it is
// reading a partially understood file is the shape §11 warns about.
func CheckManifest(file string, root *yaml.Node, manifest []byte) []problem.Problem {
	if unsupported := checkManifestSchemaVersion(file, root); unsupported != nil {
		return unsupported
	}

	problems := withReservedCapability(checkManifestBody(file, root), file, root)
	problems = append(problems, checkKind(file, root, KindProvider)...)
	problems = append(problems, checkName(file, root, "provider")...)
	return append(problems, checkOriginDigest(file, root, manifest)...)
}

// withReservedCapability layers the reserved-Capability check over the
// Manifest's own problems and drops the capability-mismatch row wherever the
// two land on one site, which is withCredentialSlots' own shape one artefact
// over: two rows pointing at one line for one cause is what that drop exists
// to prevent (§4).
//
// It stands here rather than in checkManifestBody because the subject is a
// Manifest **loaded from providers/**, which is the criterion §11 states and
// the one this function already carries — it is the half of CheckManifest the
// built-in does not run, alongside kind-mismatch and name-mismatch.
func withReservedCapability(problems []problem.Problem, file string, root *yaml.Node) []problem.Problem {
	reserved := checkCapabilityReserved(file, root)
	return append(dropCapabilityMismatchAt(problems, reserved), reserved...)
}

// dropCapabilityMismatchAt removes the capability-mismatch row standing where
// a reserved-Capability row already stands. The site is the comparison rather
// than the field, because the two rows the declared spelling can produce share
// the field capabilities: and differ only in which member of the list they
// cite — the position *is* which Capability is spoken of (§4).
//
// It is a join on node identity rather than on coordinates that happen to
// agree: both checks cite the members declaredCapabilities enumerates and the
// nodes requestCapabilities carries, so a row of each about one Capability is
// two readings of one node by construction.
//
// The drop rather than teaching capability-mismatch to pass over a reserved
// member: that check runs over the built-in too, and the built-in declaring
// shell and using it is exactly the agreement declared-equals-derived exists
// to hold it to (§4, §11).
func dropCapabilityMismatchAt(problems, reserved []problem.Problem) []problem.Problem {
	type site struct{ line, column int }
	reservedSites := map[site]bool{}
	for _, p := range reserved {
		reservedSites[site{p.Line, p.Column}] = true
	}

	kept := problems[:0]
	for _, p := range problems {
		if p.ErrorCode == CodeCapabilityMismatch && reservedSites[site{p.Line, p.Column}] {
			continue
		}
		kept = append(kept, p)
	}
	return kept
}

// checkCapabilityReserved reports capability-reserved wherever this Manifest
// reaches a Capability §12 reserves to the Providers hyper ships
// (IsReservedCapability). It reads the two spellings a Manifest reaches one
// by: the **declared** spelling, a member of the top-level capabilities: list,
// cited at that member; and the **derived** spelling, an Operation whose
// request block is that Capability's, cited at that key.
//
// **Both, and the second one is the point.** The derived spelling is where the
// power actually is — an Operation whose request block is shell: execs argv on
// the machine hyper runs on whatever the top level says — and reporting only
// the declared spelling would leave capability-mismatch to name it, as *a
// Capability an Operation names that the top level does not declare*. That is
// a row whose remedy is **declare it**, on a Manifest for which declaring it is
// the fault: a reader handed it is being told to make the file worse. Hence the
// drop above rather than two rows.
func checkCapabilityReserved(file string, root *yaml.Node) []problem.Problem {
	var problems []problem.Problem

	for _, item := range declaredCapabilities(root) {
		if !IsReservedCapability(item.Value) {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: item.Line, Column: item.Column, Field: "capabilities",
			ErrorCode: CodeCapabilityReserved,
			Message:   fmt.Sprintf("capabilities: declares %s, which is reserved to the Providers hyper ships — an Extension may never hold it", item.Value),
		})
	}

	for _, r := range requestCapabilities(root) {
		if !IsReservedCapability(r.capability) {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: r.node.Line, Column: r.node.Column,
			Field:     "operations." + r.operation + "." + r.capability,
			ErrorCode: CodeCapabilityReserved,
			Message:   fmt.Sprintf("operations.%s uses %s, which is reserved to the Providers hyper ships — an Extension may never hold it", r.operation, r.capability),
		})
	}
	return problems
}

// checkManifestBody is CheckManifest without the three checks that read a
// Manifest against where it was loaded from — reused by
// CheckBuiltinShellProvider, which authors its name outright, has no
// directory for kind-mismatch to compare against, and is the one Manifest
// entitled to the reserved Capability (§3, §11).
func checkManifestBody(file string, root *yaml.Node) []problem.Problem {
	problems := schema.Check(root, ManifestDeclaration, file)
	problems = append(problems, checkAuth(file, root)...)
	problems = append(problems, checkEnumerations(file, root)...)
	problems = append(problems, checkOperations(file, root)...)
	problems = append(problems, checkCapabilityMismatch(file, root)...)
	problems = append(problems, checkShellOnlyAuth(file, root)...)

	// env: is reserved across every artefact, a Manifest included, even
	// though a Manifest declares no credential slot of its own to carry it
	// (§3, §4) — reusing the walk withCredentialSlots layers over a
	// Target declaration's schema check for the same reason.
	problems = dropReservedEnvKey(problems)
	problems = append(problems, findReservedEnvKeys(file, root, "", nil)...)
	return problems
}

// requestCapability is one Operation's request read as the Capability it is
// written under: which Operation, which Capability, and the node whose key
// named it — the coordinate every row about that request cites.
type requestCapability struct {
	operation  string
	capability string
	node       *yaml.Node
}

// requestCapabilities is every Operation whose request names a Capability
// unambiguously, in document order — the **derived** spelling of what a
// Manifest requires.
//
// Two checks read it: capability-mismatch, which compares the set against
// capabilities:, and capability-reserved, which asks whether any member is one
// no Manifest in providers/ may hold. They read one enumeration rather than
// two walks that agree, which is operationCapability's own argument one level
// down — the day they were two walks is the day one of them reads a request
// block the other does not (§3, §4). It is also what makes the two rows about
// one Operation cite one coordinate by construction, which is the join
// dropCapabilityMismatchAt is written over.
//
// An Operation naming both http: and shell:, or neither, contributes nothing:
// it has already earned schema-mismatch from checkExactlyOneOf and names no
// Capability unambiguously. That silence is deliberate on both sides — a
// Manifest is not told it holds a Capability its own request did not name,
// and the file is refused either way.
func requestCapabilities(root *yaml.Node) []requestCapability {
	opsVal := topLevelFields(root, "operations")["operations"]
	if opsVal == nil || opsVal.Kind != yaml.MappingNode {
		return nil
	}

	var found []requestCapability
	for i := 0; i+1 < len(opsVal.Content); i += 2 {
		nameNode, opNode := opsVal.Content[i], opsVal.Content[i+1]
		if nameNode.Kind != yaml.ScalarNode || opNode.Kind != yaml.MappingNode {
			continue
		}
		capability, node := operationCapability(opNode)
		if capability == "" {
			continue
		}
		found = append(found, requestCapability{operation: nameNode.Value, capability: capability, node: node})
	}
	return found
}

// declaredCapabilities is every scalar member of the top-level capabilities:
// list, in document order — the **declared** spelling, enumerated once for the
// same two readers and for the same reason as the derived one above. The nodes
// and not the values, because which member a row cites is the only thing that
// says which Capability it is about.
func declaredCapabilities(root *yaml.Node) []*yaml.Node {
	declaredVal := topLevelFields(root, "capabilities")["capabilities"]
	if declaredVal == nil || declaredVal.Kind != yaml.SequenceNode {
		return nil
	}

	var members []*yaml.Node
	for _, item := range declaredVal.Content {
		if item.Kind == yaml.ScalarNode {
			members = append(members, item)
		}
	}
	return members
}

// checkCapabilityMismatch reads capabilities: against what hyper derives
// from every Operation's own request block — the block's key *is* the
// Capability, so this reads one key per Operation rather than inferring one
// from shape (§3). A capability declared with no Operation naming it, or
// named by an Operation the top level does not declare, is
// capability-mismatch either way (§4).
//
// It runs over every Manifest, the built-in included and with no exemption for
// the reserved Capability: declared-equals-derived is the check the whole
// extension model rests on (§11), so the one Manifest entitled to shell is
// held to it too. What capability-reserved does with the rows this earns is
// dropCapabilityMismatchAt's, one caller up.
func checkCapabilityMismatch(file string, root *yaml.Node) []problem.Problem {
	declared := map[string]bool{}
	for _, item := range declaredCapabilities(root) {
		declared[item.Value] = true
	}

	derived := map[string]bool{}
	var problems []problem.Problem
	for _, r := range requestCapabilities(root) {
		derived[r.capability] = true
		if declared[r.capability] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: r.node.Line, Column: r.node.Column,
			Field:     "operations." + r.operation + "." + r.capability,
			ErrorCode: CodeCapabilityMismatch,
			Message:   fmt.Sprintf("operations.%s uses %s, and capabilities: does not declare it", r.operation, r.capability),
		})
	}

	for _, item := range declaredCapabilities(root) {
		if derived[item.Value] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: item.Line, Column: item.Column, Field: "capabilities",
			ErrorCode: CodeCapabilityMismatch,
			Message:   fmt.Sprintf("capabilities: declares %s, and no Operation's request block names it", item.Value),
		})
	}
	return problems
}

// operationCapability is the one Capability an Operation's request is
// written under, and the node that names it: the request block's own key
// *is* the Capability, so this reads one key rather than inferring one from
// shape (§3, §12). It answers "" where the Operation names both http: and
// shell:, or neither — a request that has already earned schema-mismatch
// from checkExactlyOneOf and names no Capability unambiguously.
//
// One derivation rather than two that agree: the Capability check refuses a
// Manifest's capabilities: for disagreeing with it, and §9's derived block
// reports it, and the day they were two walks is the day one of them reads
// a request block the other does not (§4, §9).
func operationCapability(op *yaml.Node) (string, *yaml.Node) {
	fields := topLevelFields(op, "http", "shell")
	httpVal, shellVal := fields["http"], fields["shell"]
	switch {
	case httpVal != nil && shellVal == nil:
		return "http", httpVal
	case shellVal != nil && httpVal == nil:
		return "shell", shellVal
	default:
		return "", nil
	}
}

// checkShellOnlyAuth reports manifest-inconsistent on a Manifest declaring
// only the shell Capability while carrying an auth: block — auth is a
// property of reaching a host, and shell reaches none (§3, §4).
func checkShellOnlyAuth(file string, root *yaml.Node) []problem.Problem {
	fields := topLevelFields(root, "capabilities", "auth")
	capsVal, authVal := fields["capabilities"], fields["auth"]
	if capsVal == nil || capsVal.Kind != yaml.SequenceNode || authVal == nil {
		return nil
	}
	if len(capsVal.Content) != 1 || capsVal.Content[0].Kind != yaml.ScalarNode || capsVal.Content[0].Value != "shell" {
		return nil
	}
	line, column := position(authVal)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: "auth",
		ErrorCode: CodeManifestInconsistent,
		Message:   "a Provider declaring only the shell Capability carries no auth: block — auth is a property of reaching a host",
	}}
}

// authOwnedHeaderName reads root's auth: scheme and returns the header
// position it owns — header:'s own name: parameter, or basic:'s fixed
// Authorization (§13) — the position an ordinary headers: entry may not
// also name (manifest-inconsistent, §4). It returns "" where auth: is
// absent or its scheme's own shape is too malformed to say.
func authOwnedHeaderName(root *yaml.Node) string {
	authVal := topLevelFields(root, "auth")["auth"]
	if authVal == nil || authVal.Kind != yaml.MappingNode {
		return ""
	}
	fields := topLevelFields(authVal, authSchemes...)
	if headerVal := fields["header"]; headerVal != nil {
		if headerVal.Kind != yaml.MappingNode {
			return ""
		}
		if nameVal := topLevelFields(headerVal, "name")["name"]; nameVal != nil && nameVal.Kind == yaml.ScalarNode {
			return nameVal.Value
		}
		return ""
	}
	if fields["basic"] != nil {
		return "Authorization"
	}
	return ""
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
	ownedHeader := authOwnedHeaderName(root)

	var problems []problem.Problem
	for i := 0; i+1 < len(opsVal.Content); i += 2 {
		nameNode, opNode := opsVal.Content[i], opsVal.Content[i+1]
		if nameNode.Kind != yaml.ScalarNode {
			continue
		}
		problems = append(problems, checkOneOperation(file, "operations."+nameNode.Value, opNode, enumNames, ownedHeader)...)
	}
	return problems
}

// checkOneOperation validates one operations: entry: its own flat schema,
// exactly one request block between http: and shell:, the input-schema
// subset, the three Patterns, and the record: projection's paths and holes
// (§3, §4, §12).
func checkOneOperation(file, field string, op *yaml.Node, enumNames map[string]bool, ownedHeader string) []problem.Problem {
	problems := schema.CheckAt(op, operationDeclaration, field, file)
	if op == nil || op.Kind != yaml.MappingNode {
		return problems
	}

	problems = append(problems, checkExactlyOneOf(file, field, op, []string{"http", "shell"})...)

	fields := topLevelFields(op, "kind", "repeatability", "concurrency", "http", "shell", "input", "patterns", "record")
	inputProps := inputPropertyNames(fields["input"])
	inputTypes := inputPropertyTypes(fields["input"])
	isHTTP := fields["http"] != nil
	isShell := fields["shell"] != nil

	if httpVal := fields["http"]; httpVal != nil && httpVal.Kind == yaml.MappingNode {
		problems = append(problems, checkHTTPRequest(file, field+".http", httpVal, enumNames, inputProps, inputTypes, ownedHeader)...)
	}
	if inputVal := fields["input"]; inputVal != nil && inputVal.Kind == yaml.MappingNode {
		problems = append(problems, checkInputSchema(file, field+".input", inputVal)...)
	}
	if patternsVal := fields["patterns"]; patternsVal != nil && patternsVal.Kind == yaml.MappingNode {
		problems = append(problems, checkPatterns(file, field+".patterns", patternsVal)...)
	}
	if recordVal := fields["record"]; recordVal != nil && recordVal.Kind == yaml.MappingNode {
		problems = append(problems, checkRecord(file, field+".record", recordVal, inputProps, inputTypes)...)
	}

	// The remaining cross-checks read an Operation's own top-level keys
	// against each other rather than against a nested block's shape, so
	// they run unconditionally over fields rather than being gated on one
	// sub-block's own Kind, the way the four calls above are (§3, §4).
	problems = append(problems, checkKindProjection(file, field, fields)...)
	problems = append(problems, checkPaginationOverRequirement(file, field, fields)...)
	problems = append(problems, checkSkipIfRecordedIdentity(file, field, fields, isShell)...)
	if isHTTP != isShell {
		problems = append(problems, checkInputReachability(file, field, fields, isShell)...)
	}
	return problems
}

// checkKindProjection reports manifest-inconsistent on the four ways an
// Operation's own Kind disagrees with what it projects or what its
// Repeatability or its concurrency would have to mean (§3, §4): a read or
// mutate carrying no record:, a destroy carrying one, skip-if-recorded
// declared on an Operation that is not a mutate (ADR-0037), and a
// concurrency: limit declared on an Operation that is not a read
// (ADR-0045). Each earns no code of its own, pointing a reader at one file,
// one Operation, and two adjacent keys instead.
func checkKindProjection(file, field string, fields map[string]*yaml.Node) []problem.Problem {
	kindVal := fields["kind"]
	if kindVal == nil || kindVal.Kind != yaml.ScalarNode {
		return nil
	}
	kind := kindVal.Value
	recordVal := fields["record"]

	var problems []problem.Problem
	switch {
	case (kind == "read" || kind == "mutate") && recordVal == nil:
		problems = append(problems, problem.Problem{
			File: file, Line: kindVal.Line, Column: kindVal.Column, Field: field,
			ErrorCode: CodeManifestInconsistent,
			Message:   fmt.Sprintf("kind: %s carries no record: — record: is mandatory on a read and a mutate", kind),
		})
	case kind == "destroy" && recordVal != nil:
		line, column := position(recordVal)
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".record",
			ErrorCode: CodeManifestInconsistent,
			Message:   "kind: destroy carries a record: — record: is forbidden on a destroy",
		})
	}

	if repVal := fields["repeatability"]; repVal != nil && repVal.Kind == yaml.ScalarNode &&
		repVal.Value == "skip-if-recorded" && kind != "mutate" {
		problems = append(problems, problem.Problem{
			File: file, Line: repVal.Line, Column: repVal.Column, Field: field + ".repeatability",
			ErrorCode: CodeManifestInconsistent,
			Message:   "repeatability: skip-if-recorded is declared on an Operation that is not a mutate",
		})
	}

	if concVal := fields["concurrency"]; concVal != nil && kind != "read" {
		problems = append(problems, problem.Problem{
			File: file, Line: concVal.Line, Column: concVal.Column, Field: field + ".concurrency",
			ErrorCode: CodeManifestInconsistent,
			Message:   "concurrency: is declared on an Operation that is not a read",
		})
	}
	return problems
}

// checkPaginationOverRequirement reports manifest-inconsistent on a
// pagination Pattern declared on an Operation whose record: carries no
// over: — pagination walks the collection record.over names, and no
// collection means nothing for it to walk (§3, §4).
func checkPaginationOverRequirement(file, field string, fields map[string]*yaml.Node) []problem.Problem {
	patternsVal := fields["patterns"]
	if patternsVal == nil || patternsVal.Kind != yaml.MappingNode {
		return nil
	}
	paginationVal := topLevelFields(patternsVal, "pagination")["pagination"]
	if paginationVal == nil {
		return nil
	}
	if overVal := topLevelFields(fields["record"], "over")["over"]; overVal != nil {
		return nil
	}
	line, column := position(paginationVal)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: field + ".patterns.pagination",
		ErrorCode: CodeManifestInconsistent,
		Message:   "patterns.pagination is declared and record: carries no over:",
	}}
}

// checkSkipIfRecordedIdentity reports manifest-inconsistent on a
// skip-if-recorded Operation whose identity: resolves only from the
// response. Its test reads the head of the series before deciding whether
// to make the call, so identity: must resolve before the call: a template
// hole does, and so does $.command on a shell Operation, sitting in the
// response object precisely because it is a fact about the call rather than
// about the answer; a response path anywhere else names a value that exists
// only once the call has gone out (§3, §4, §12, ADR-0056).
func checkSkipIfRecordedIdentity(file, field string, fields map[string]*yaml.Node, isShell bool) []problem.Problem {
	repVal := fields["repeatability"]
	if repVal == nil || repVal.Kind != yaml.ScalarNode || repVal.Value != "skip-if-recorded" {
		return nil
	}
	identityVal := topLevelFields(fields["record"], "identity")["identity"]
	if identityVal == nil || identityVal.Kind != yaml.ScalarNode || !strings.HasPrefix(identityVal.Value, "$") {
		return nil
	}
	if isShell && identityVal.Value == "$.command" {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: identityVal.Line, Column: identityVal.Column, Field: field + ".record.identity",
		ErrorCode: CodeManifestInconsistent,
		Message:   fmt.Sprintf("%q resolves only from the response — a skip-if-recorded test must resolve before the call", identityVal.Value),
	}}
}

// checkInputReachability reports manifest-inconsistent on every input this
// Operation declares that no position of its request reaches: no ordinary
// hole, no host-input:, and — on a shell Operation, the one Capability
// whose request has no shape of its own to carry a hole — not the argv the
// shell Capability names, the Operation input named command by convention
// rather than by any position (§3, §4). It is called only where the
// Operation names exactly one of http:/shell:, an ambiguous or absent
// request having already earned schema-mismatch and reaching no position at
// all.
func checkInputReachability(file, field string, fields map[string]*yaml.Node, isShell bool) []problem.Problem {
	propsVal := topLevelFields(fields["input"], "properties")["properties"]
	if propsVal == nil || propsVal.Kind != yaml.MappingNode || len(propsVal.Content) == 0 {
		return nil
	}

	reached := map[string]bool{}
	if isShell {
		reached[ShellCommandInput] = true
	} else {
		collectReachedNames(fields["http"], reached)
	}

	var problems []problem.Problem
	for i := 0; i+1 < len(propsVal.Content); i += 2 {
		key := propsVal.Content[i]
		if key.Kind != yaml.ScalarNode || reached[key.Value] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: key.Line, Column: key.Column, Field: field + ".input.properties." + key.Value,
			ErrorCode: CodeManifestInconsistent,
			Message:   fmt.Sprintf("input %q is declared and no position of this Operation's request reaches it", key.Value),
		})
	}
	return problems
}

// collectReachedNames marks, in reached, every input name an http: block's
// ordinary positions reach: every hole in method:, path:, query: and
// headers: values, every hole in body:, and host-input:'s own value (§3,
// §4). host: is deliberately excluded — it is Capability-relevant and never
// resolves to an Operation input (§12).
func collectReachedNames(httpVal *yaml.Node, reached map[string]bool) {
	if httpVal == nil || httpVal.Kind != yaml.MappingNode {
		return
	}
	fields := topLevelFields(httpVal, "method", "path", "query", "headers", "body", "host-input")
	for _, key := range []string{"method", "path"} {
		if v := fields[key]; v != nil && v.Kind == yaml.ScalarNode {
			collectHoleNames(v.Value, reached)
		}
	}
	for _, key := range []string{"query", "headers"} {
		if m := fields[key]; m != nil && m.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(m.Content); i += 2 {
				if val := m.Content[i+1]; val.Kind == yaml.ScalarNode {
					collectHoleNames(val.Value, reached)
				}
			}
		}
	}
	if b := fields["body"]; b != nil {
		collectBodyHoleNames(b, reached)
	}
	if hi := fields["host-input"]; hi != nil && hi.Kind == yaml.ScalarNode {
		reached[hi.Value] = true
	}
}

// collectHoleNames marks every hole s carries in reached.
func collectHoleNames(s string, reached map[string]bool) {
	for _, m := range holePattern.FindAllStringSubmatch(s, -1) {
		reached[m[1]] = true
	}
}

// collectBodyHoleNames walks a body: value tree marking every hole its
// values carry in reached. A mapping key is never a legal hole position
// (hole-illegal catches one that tries), so only values are read here
// (§3, §4).
func collectBodyHoleNames(node *yaml.Node, reached map[string]bool) {
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			collectBodyHoleNames(node.Content[i+1], reached)
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			collectBodyHoleNames(item, reached)
		}
	case yaml.ScalarNode:
		collectHoleNames(node.Value, reached)
	}
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
func checkHTTPRequest(file, field string, node *yaml.Node, enumNames, inputProps map[string]bool, inputTypes map[string]string, ownedHeader string) []problem.Problem {
	problems := schema.CheckAt(node, httpRequestDeclaration, field, file)
	fields := topLevelFields(node, "method", "host", "path", "query", "headers", "body", "host-input")

	if methodVal := fields["method"]; methodVal != nil && methodVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkOrdinaryHoles(file, field+".method", methodVal, inputProps, inputTypes)...)
	}
	if hostVal := fields["host"]; hostVal != nil && hostVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkCapabilityHoles(file, field+".host", hostVal, enumNames)...)
	}
	if pathVal := fields["path"]; pathVal != nil && pathVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkOrdinaryHoles(file, field+".path", pathVal, inputProps, inputTypes)...)
		problems = append(problems, checkPathDelimiters(file, field+".path", pathVal)...)
	}
	problems = append(problems, checkStringMapping(file, field+".query", fields["query"], inputProps, inputTypes)...)
	problems = append(problems, checkStringMapping(file, field+".headers", fields["headers"], inputProps, inputTypes)...)
	problems = append(problems, checkHeadersReserved(file, field+".headers", fields["headers"])...)
	problems = append(problems, checkHeadersOwnedPosition(file, field+".headers", fields["headers"], ownedHeader)...)
	problems = append(problems, checkBody(file, field+".body", fields["body"], inputProps, inputTypes)...)

	if hostInputVal := fields["host-input"]; hostInputVal != nil && hostInputVal.Kind == yaml.ScalarNode && !inputProps[hostInputVal.Value] {
		problems = append(problems, problem.Problem{
			File: file, Line: hostInputVal.Line, Column: hostInputVal.Column, Field: field + ".host-input",
			ErrorCode: CodeManifestInconsistent,
			Message:   fmt.Sprintf("host-input: %s names no input this Operation declares", hostInputVal.Value),
		})
	}
	return problems
}

// checkPathDelimiters reports manifest-inconsistent on a path: carrying a
// ? or a #: the two gen-delims that end a path in RFC 3986 and that neither
// url.URL nor hyper will read as one, since path: is written as text and
// hyper does the percent-encoding — so a query written there goes out as
// %3F inside the path and reaches nothing (§3, §4, ADR-0107, issue #229).
//
// It is manifest-inconsistent rather than a code of its own because it is
// the same shape the other eleven here have: one file, one Operation, and
// two adjacent keys — the value is in path: and it belongs in query:,
// which is the key beside it. The # has no key to move to and is refused
// with it anyway: a fragment is never transmitted, so admitting the one
// character an author cannot use while refusing the one they can move
// would be the wrong half of the pair.
//
// A ? wins over a #, and one row is the whole of it. In a URI the ? opens
// the query and everything after it is inside what the author meant as one,
// so a second row on the same line would name a second fault that is not
// there, and the edit that fixes the first fixes both.
//
// Holes are elided before the text is read. A hole's text is a name, and a
// name is checked as a name — by checkOrdinaryHoles, on the same node — so
// reading one as path text would put two rows on one line for one fault.
func checkPathDelimiters(file, field string, node *yaml.Node) []problem.Problem {
	text := holePattern.ReplaceAllString(node.Value, "")
	var message string
	switch {
	case strings.Contains(text, "?"):
		message = "path: carries a ? — a query is written in the query: key beside it, and a ? here is escaped into the path rather than opening one"
	case strings.Contains(text, "#"):
		message = "path: carries a # — a fragment is never sent to a server, and a # here is escaped into the path rather than becoming one"
	default:
		return nil
	}
	return []problem.Problem{{
		File: file, Line: node.Line, Column: node.Column, Field: field,
		ErrorCode: CodeManifestInconsistent,
		Message:   message,
	}}
}

// checkHeadersReserved reports header-reserved on every headers: entry
// naming one of the five headers hyper computes for itself, compared
// case-insensitively (§4, §12).
func checkHeadersReserved(file, field string, node *yaml.Node) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var problems []problem.Problem
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		problems = append(problems, reservedHeaderProblem(file, field+"."+key.Value, key)...)
	}
	return problems
}

// reservedHeaderProblem is the one check behind header-reserved's two
// writers — an Auth scheme's name: parameter and an ordinary headers:
// entry — reporting the same fault under the same code and message
// wherever nameVal names one of the five headers hyper computes for itself,
// compared case-insensitively (§4, §12).
func reservedHeaderProblem(file, field string, nameVal *yaml.Node) []problem.Problem {
	if nameVal.Kind != yaml.ScalarNode || !reservedHeaders[strings.ToLower(nameVal.Value)] {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: nameVal.Line, Column: nameVal.Column, Field: field,
		ErrorCode: CodeHeaderReserved,
		Message:   fmt.Sprintf("%s: is one of the five headers hyper computes for itself", nameVal.Value),
	}}
}

// checkHeadersOwnedPosition reports manifest-inconsistent on a headers:
// entry naming the position this Manifest's own Auth scheme owns —
// header:'s name: parameter, or basic:'s fixed Authorization — compared
// case-insensitively: a second writer there would disagree with what hyper
// places, auth being suppressed by the position it occupies rather than by
// scanning a rendering for something that looks like one (§3, §4).
func checkHeadersOwnedPosition(file, field string, node *yaml.Node, ownedHeader string) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode || ownedHeader == "" {
		return nil
	}
	var problems []problem.Problem
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Kind != yaml.ScalarNode || !strings.EqualFold(key.Value, ownedHeader) {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: key.Line, Column: key.Column, Field: field + "." + key.Value,
			ErrorCode: CodeManifestInconsistent,
			Message:   fmt.Sprintf("%s: takes the request position this Manifest's Auth scheme already owns", key.Value),
		})
	}
	return problems
}

// checkStringMapping reads a query: or headers: mapping's members, which
// are always name to string (§3): a non-scalar value is schema-mismatch,
// and a scalar's holes are checked as an ordinary position's are.
func checkStringMapping(file, field string, node *yaml.Node, inputProps map[string]bool, inputTypes map[string]string) []problem.Problem {
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
		problems = append(problems, checkOrdinaryHoles(file, childField, val, inputProps, inputTypes)...)
	}
	return problems
}

// checkBody walks a body: value tree — the one position in any artefact
// hyper holds no schema for (§3). A literal scalar is left untyped by this
// check on purpose, carrying its YAML 1.2 core type onto the wire; what
// this still checks is the two rules that hold regardless of the API's own
// shape: a hole fills a value only, never a mapping key, and a hole in a
// value resolves only against this Operation's own input.
func checkBody(file, field string, node *yaml.Node, inputProps map[string]bool, inputTypes map[string]string) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	return checkBodyValue(file, field, node, inputProps, inputTypes)
}

func checkBodyValue(file, field string, node *yaml.Node, inputProps map[string]bool, inputTypes map[string]string) []problem.Problem {
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
			problems = append(problems, checkBodyValue(file, childField, val, inputProps, inputTypes)...)
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			problems = append(problems, checkBodyValue(file, fmt.Sprintf("%s[%d]", field, i), item, inputProps, inputTypes)...)
		}
	case yaml.ScalarNode:
		problems = append(problems, checkOrdinaryHoles(file, field, node, inputProps, inputTypes)...)
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
// (§3, §12). One naming an input the same file declares object or array is
// manifest-inconsistent instead: the name exists, so the fault is a
// Manifest disagreeing with itself rather than a name resolving to nothing
// — a hole fills a scalar position, and a whole object is no more
// interpolable than it is referenceable (§3, §4, ADR-0078).
func checkOrdinaryHoles(file, field string, node *yaml.Node, inputProps map[string]bool, inputTypes map[string]string) []problem.Problem {
	var problems []problem.Problem
	for _, m := range holePattern.FindAllStringSubmatch(node.Value, -1) {
		name := m[1]
		if !inputProps[name] {
			problems = append(problems, problem.Problem{
				File: file, Line: node.Line, Column: node.Column, Field: field,
				ErrorCode: CodeHoleIllegal,
				Message:   fmt.Sprintf("{%s} names no input this Operation declares", name),
			})
			continue
		}
		if t := inputTypes[name]; t == "object" || t == "array" {
			problems = append(problems, problem.Problem{
				File: file, Line: node.Line, Column: node.Column, Field: field,
				ErrorCode: CodeManifestInconsistent,
				Message:   fmt.Sprintf("{%s} names an input declared %s — a hole fills a scalar position only", name, t),
			})
		}
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

// inputPropertyTypes reads the top-level type: an input: schema declares
// for each of its properties, keyed by property name, omitted where the
// property's own schema omits type: — the set checkOrdinaryHoles compares a
// hole's name against, object and array being the two a hole may never
// fill (§3, §4, §12).
func inputPropertyTypes(inputVal *yaml.Node) map[string]string {
	types := map[string]string{}
	if inputVal == nil || inputVal.Kind != yaml.MappingNode {
		return types
	}
	propsVal := topLevelFields(inputVal, "properties")["properties"]
	if propsVal == nil || propsVal.Kind != yaml.MappingNode {
		return types
	}
	for i := 0; i+1 < len(propsVal.Content); i += 2 {
		key, val := propsVal.Content[i], propsVal.Content[i+1]
		if key.Kind != yaml.ScalarNode || val.Kind != yaml.MappingNode {
			continue
		}
		if typeVal := topLevelFields(val, "type")["type"]; typeVal != nil && typeVal.Kind == yaml.ScalarNode {
			types[key.Value] = typeVal.Value
		}
	}
	return types
}

// operationInfoFromNode reads the five facts a Step checked against this
// Operation needs off the Operation's own node (§3, §4, issue #94): whether
// its request is the shell Capability, its input: schema's own property
// names, types and enums, its record: cardinality and field names — nil
// where it declares no record: at all, a destroy carrying none by
// construction (checkKindProjection) — and its own secret: field names, the
// set a predicate's own field: is checked against (§12, issue #97). It is
// read once per Manifest pass, alongside the rest of ProviderInfo, rather
// than reparsed per Step that names this Operation.
func operationInfoFromNode(op *yaml.Node) OperationInfo {
	fields := topLevelFields(op, "kind", "shell", "http", "input", "record", "repeatability", "secret")
	info := OperationInfo{IsShell: fields["shell"] != nil, Inputs: map[string]InputInfo{}}
	if httpVal := fields["http"]; httpVal != nil && httpVal.Kind == yaml.MappingNode {
		httpFields := topLevelFields(httpVal, "host", "host-input")
		if hostVal := httpFields["host"]; hostVal != nil && hostVal.Kind == yaml.ScalarNode {
			info.HostTemplate = hostVal.Value
		}
		if hostInputVal := httpFields["host-input"]; hostInputVal != nil && hostInputVal.Kind == yaml.ScalarNode {
			info.HostInput = hostInputVal.Value
		}
	}
	if kindVal := fields["kind"]; kindVal != nil && kindVal.Kind == yaml.ScalarNode {
		info.Kind = kindVal.Value
	}
	if repVal := fields["repeatability"]; repVal != nil && repVal.Kind == yaml.ScalarNode {
		info.Repeatability = repVal.Value
	}
	if secretVal := fields["secret"]; secretVal != nil && secretVal.Kind == yaml.SequenceNode {
		info.HasSecret = len(secretVal.Content) > 0
		info.SecretFields = map[string]bool{}
		for _, item := range secretVal.Content {
			if item.Kind == yaml.ScalarNode {
				info.SecretFields[item.Value] = true
			}
		}
	}

	if inputVal := fields["input"]; inputVal != nil && inputVal.Kind == yaml.MappingNode {
		types := inputPropertyTypes(inputVal)
		enums := inputPropertyEnums(inputVal)
		for name := range inputPropertyNames(inputVal) {
			info.Inputs[name] = InputInfo{Type: types[name], Enum: enums[name]}
		}
	}

	if recordVal := fields["record"]; recordVal != nil && recordVal.Kind == yaml.MappingNode {
		recordFields := topLevelFields(recordVal, "over", "fields", "identity")
		info.HasSeries = recordFields["over"] != nil
		if identityVal := recordFields["identity"]; identityVal != nil && identityVal.Kind == yaml.ScalarNode {
			info.Identity = identityVal.Value
		}
		info.RecordFields = map[string]bool{}
		if fieldsVal := recordFields["fields"]; fieldsVal != nil && fieldsVal.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(fieldsVal.Content); i += 2 {
				if key := fieldsVal.Content[i]; key.Kind == yaml.ScalarNode {
					info.RecordFields[key.Value] = true
				}
			}
		}
	}
	return info
}

// inputPropertyEnums reads the top-level enum: an input: schema declares
// for each of its properties, keyed by property name, omitted where the
// property's own schema omits enum: — the set a Step's args: value is
// checked against, on inputPropertyTypes's own rule (§3, §4, issue #94).
func inputPropertyEnums(inputVal *yaml.Node) map[string][]string {
	enums := map[string][]string{}
	if inputVal == nil || inputVal.Kind != yaml.MappingNode {
		return enums
	}
	propsVal := topLevelFields(inputVal, "properties")["properties"]
	if propsVal == nil || propsVal.Kind != yaml.MappingNode {
		return enums
	}
	for i := 0; i+1 < len(propsVal.Content); i += 2 {
		key, val := propsVal.Content[i], propsVal.Content[i+1]
		if key.Kind != yaml.ScalarNode || val.Kind != yaml.MappingNode {
			continue
		}
		enumVal := topLevelFields(val, "enum")["enum"]
		if enumVal == nil || enumVal.Kind != yaml.SequenceNode {
			continue
		}
		var values []string
		for _, item := range enumVal.Content {
			if item.Kind == yaml.ScalarNode {
				values = append(values, item.Value)
			}
		}
		enums[key.Value] = values
	}
	return enums
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
	problems := checkExactlyOneOf(file, "auth", authVal, authSchemes)
	fields := topLevelFields(authVal, authSchemes...)
	if headerVal := fields["header"]; headerVal != nil {
		problems = append(problems, schema.CheckAt(headerVal, authHeaderDeclaration, "auth.header", file)...)
		problems = append(problems, checkAuthHoles(file, "auth.header", headerVal)...)
		if headerVal.Kind == yaml.MappingNode {
			if nameVal := topLevelFields(headerVal, "name")["name"]; nameVal != nil {
				problems = append(problems, reservedHeaderProblem(file, "auth.header.name", nameVal)...)
			}
		}
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

// checkPredicate validates one polling Pattern until: predicate: the shape
// and operand types checkPredicateCore reads regardless of root (§12, issue
// #97), and this root's own rule — field: is a path in the grammar, written
// without the root marker, a response having paths and no declared names
// (§12). It carries no step: — a polling Pattern's until: roots at the
// response object in hand rather than at an earlier Step's Record.
func checkPredicate(file, field string, node *yaml.Node) []problem.Problem {
	problems, fieldNameVal, _ := checkPredicateCore(file, field, node, false)
	if fieldNameVal != nil {
		problems = append(problems, checkFieldPathNoRoot(file, field+".field", fieldNameVal)...)
	}
	return problems
}

// checkRecord validates a record: block's own schema, then its paths: an
// identity: that is either a response path or a template hole, an over:
// that is always a response path, and every fields: entry, a response path
// (§3, §12).
func checkRecord(file, field string, node *yaml.Node, inputProps map[string]bool, inputTypes map[string]string) []problem.Problem {
	problems := schema.CheckAt(node, recordDeclaration, field, file)
	if node == nil || node.Kind != yaml.MappingNode {
		return problems
	}
	fields := topLevelFields(node, "identity", "fields", "over")

	if fields["identity"] == nil {
		line, column := position(node)
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeIdentityUndeclared,
			Message:   "record: projects a Record and declares no identity: for it",
		})
	}

	if identityVal := fields["identity"]; identityVal != nil && identityVal.Kind == yaml.ScalarNode {
		problems = append(problems, checkIdentity(file, field+".identity", identityVal, inputProps, inputTypes)...)
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
func checkIdentity(file, field string, node *yaml.Node, inputProps map[string]bool, inputTypes map[string]string) []problem.Problem {
	switch {
	case strings.HasPrefix(node.Value, "$"):
		return checkPathValue(file, field, node)
	case strings.HasPrefix(node.Value, "{"):
		return checkOrdinaryHoles(file, field, node, inputProps, inputTypes)
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
