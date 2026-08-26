package artefact

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// A case whose Manifest is written with shell: request blocks runs
// checkManifestBody rather than CheckManifest, and what it asserts is why: the
// shell Capability is reserved to the Providers hyper ships, so a Manifest of
// that shape is the built-in's and not a file in providers/ (§11, §12,
// capability-reserved). CheckManifest is the providers/ file's checks —
// kind-mismatch, name-mismatch and the reserved Capability — and
// checkManifestBody is the Manifest's own oracle, which is what a case about
// an input schema or a record identity is asking about.

// mustNoCode fails if got carries any problem of code.
func mustNoCode(t *testing.T, got []problem.Problem, code string) {
	t.Helper()
	for _, p := range got {
		if p.ErrorCode == code {
			t.Fatalf("got %+v, want no %s problem", got, code)
		}
	}
}

// countCode is how many problems of code got carries — the assertion a check
// firing at more than one site needs, where mustCode answers only the first.
func countCode(got []problem.Problem, code string) int {
	return len(fieldsOfCode(got, code))
}

// fieldsOfCode is the field each problem of code cites, in the order they were
// reported. A check firing at more than one site is asserted through this
// rather than by indexing got, so what the assertion reads is which sites were
// cited and not where a caller happened to append them.
func fieldsOfCode(got []problem.Problem, code string) []string {
	var fields []string
	for _, p := range got {
		if p.ErrorCode == code {
			fields = append(fields, p.Field)
		}
	}
	return fields
}

// cloudflareDNS and uptime are §3's two worked Manifests, byte for byte
// (issue #91's acceptance criteria: "§3's two worked Manifests —
// cloudflare-dns and uptime — both check clean").
const cloudflareDNS = `kind: provider
provider: cloudflare-dns
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  create_dns_record:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records
      body: {name: "{name}", type: "{type}", content: "{content}"}
    input:
      type: object
      properties:
        zone_id: {type: string}
        name: {type: string}
        type: {type: string, enum: [A, AAAA, CNAME]}
        content: {type: string}
    record:
      identity: "{name}"
      fields:
        id: $.body.result.id
        name: $.body.result.name
        created_on: $.body.result.created_on
  list_dns_records:
    kind: read
    repeatability: repeatable
    deadline: 30s
    concurrency: 4
    http:
      method: GET
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records
      query: {per_page: "100"}
    input:
      type: object
      properties:
        zone_id: {type: string}
    patterns:
      pagination:
        cursor: {from: $.body.result_info.cursor, into: {query: cursor}}
      retry: {attempts: 3}
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id, name: $.name, created_on: $.created_on}
  delete_dns_record:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http:
      method: DELETE
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records/{record_id}
    input:
      type: object
      properties:
        zone_id: {type: string}
        record_id: {type: string}
`

const uptime = `kind: provider
provider: uptime
schema-version: 1
class: local
capabilities: [http]
operations:
  check_http:
    kind: read
    deadline: 10s
    http:
      method: GET
      host: "{from-target}"
      path: /
      host-input: host
    input:
      type: object
      properties:
        host: {type: string}
    record:
      identity: $.host
      fields:
        host: $.host
        status: $.status
        days_left: $.tls.days_left
`

func TestCheckManifest_CloudflareDNSIsClean(t *testing.T) {
	mustNone(t, CheckManifest("providers/cloudflare-dns.yaml", parse(t, cloudflareDNS)))
}

func TestCheckManifest_UptimeIsClean(t *testing.T) {
	mustNone(t, CheckManifest("providers/uptime.yaml", parse(t, uptime)))
}

func TestCheckManifest_KindMismatch(t *testing.T) {
	doc := "kind: definition\nprovider: uptime\nschema-version: 1\nclass: local\ncapabilities: [http]\noperations: {}\n"
	got := CheckManifest("providers/uptime.yaml", parse(t, doc))
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckManifest_NameMismatch(t *testing.T) {
	got := CheckManifest("providers/other.yaml", parse(t, uptime))
	p := mustCode(t, got, CodeNameMismatch)
	if p.Field != "provider" {
		t.Errorf("Field = %q, want provider", p.Field)
	}
}

func TestCheckManifest_TopLevelSchemaAdmitsExactlyTheDocumentedKeys(t *testing.T) {
	doc := uptime + "origin: {ref: registry/uptime@1, digest: sha256:00}\n"
	mustNone(t, CheckManifest("providers/uptime.yaml", parse(t, doc)))

	got := CheckManifest("providers/uptime.yaml", parse(t, uptime+"extra: 1\n"))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "extra" {
		t.Errorf("Field = %q, want extra", p.Field)
	}
}

func TestCheckManifest_DeadlineAbsentIsSchemaMismatch(t *testing.T) {
	doc := `kind: provider
provider: uptime
schema-version: 1
class: local
capabilities: [http]
operations:
  check_http:
    kind: read
    http:
      method: GET
      host: "{from-target}"
      path: /
`
	got := CheckManifest("providers/uptime.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "operations.check_http.deadline" {
		t.Errorf("Field = %q, want operations.check_http.deadline", p.Field)
	}
}

func TestCheckManifest_OpaqueIsNotAWritableKey(t *testing.T) {
	doc := `kind: provider
provider: shell-ish
schema-version: 1
class: local
capabilities: [shell]
operations:
  read:
    kind: read
    deadline: 1h
    opaque: true
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
`
	got := CheckManifest("providers/shell-ish.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "operations.read.opaque" {
		t.Errorf("Field = %q, want operations.read.opaque", p.Field)
	}
}

func TestCheckManifest_NeitherHTTPNorShellIsSchemaMismatch(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: []
operations:
  noop:
    kind: read
    deadline: 1h
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "operations.noop" {
		t.Errorf("Field = %q, want operations.noop", p.Field)
	}
}

func TestCheckManifest_BothHTTPAndShellIsSchemaMismatch(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http, shell]
operations:
  both:
    kind: read
    deadline: 1h
    shell: {}
    http:
      method: GET
      host: "{from-target}"
      path: /
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "operations.both" {
		t.Errorf("Field = %q, want operations.both", p.Field)
	}
}

func TestCheckManifest_ShellBlockIsWhollyEmpty(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [shell]
operations:
  run:
    kind: read
    deadline: 1h
    shell: {command: [echo, hi]}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "operations.run.shell.command" {
		t.Errorf("Field = %q, want operations.run.shell.command", p.Field)
	}
}

func TestCheckManifestBody_InputSchemaOutsideTheSubsetIsUnsupported(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema string
	}{
		{"ref", `{"$ref": "#/definitions/x"}`},
		{"allOf", `{allOf: [{type: string}]}`},
		{"oneOf", `{oneOf: [{type: string}]}`},
		{"ifThenElse", `{if: {type: string}, then: {type: string}}`},
		{"required", `{type: object, properties: {a: {type: string}}, required: [a]}`},
		{"const", `{type: string, const: fixed}`},
		{"additionalProperties", `{type: object, additionalProperties: false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := "kind: provider\nprovider: broken\nschema-version: 1\nclass: local\ncapabilities: [shell]\noperations:\n  run:\n    kind: read\n    deadline: 1h\n    shell: {}\n    input: " + tc.schema + "\n"
			got := checkManifestBody("providers/broken.yaml", parse(t, doc))
			mustCode(t, got, CodeSchemaUnsupported)
		})
	}
}

func TestCheckManifestBody_InputSchemaSubsetNestsCleanly(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [shell]
operations:
  run:
    kind: read
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command:
          type: array
          items:
            type: string
            enum: [a, b]
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
`
	mustNone(t, checkManifestBody("providers/broken.yaml", parse(t, doc)))
}

func TestCheckManifest_HoleInAuthParametersIsIllegal(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: "{header_name}", prefix: "Bearer "}
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    input:
      type: object
      properties:
        header_name: {type: string}
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeHoleIllegal)
	if p.Field != "auth.header.name" {
		t.Errorf("Field = %q, want auth.header.name", p.Field)
	}
}

func TestCheckManifest_HoleInABodyMappingKeyIsIllegal(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
operations:
  noop:
    kind: mutate
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: /widgets
      body:
        "{name}": literal
    input:
      type: object
      properties:
        name: {type: string}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeHoleIllegal)
	if p.Field != "operations.noop.http.body.{name}" {
		t.Errorf("Field = %q, want operations.noop.http.body.{name}", p.Field)
	}
}

func TestCheckManifest_HostHoleMustResolveToEnumerationOrFromTarget(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: aws
capabilities: [http]
operations:
  list:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "s3.{bucket}.amazonaws.com"
      path: /
    input:
      type: object
      properties:
        bucket: {type: string}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeHoleIllegal)
	if p.Field != "operations.list.http.host" {
		t.Errorf("Field = %q, want operations.list.http.host", p.Field)
	}
}

func TestCheckManifest_HostHoleResolvingToEnumerationIsLegal(t *testing.T) {
	doc := `kind: provider
provider: aws-s3
schema-version: 1
class: aws
capabilities: [http]
enumerations:
  region: [us-east-1, eu-central-1]
operations:
  list_buckets:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "s3.{region}.amazonaws.com"
      path: /
      host-input: endpoint
    input:
      type: object
      properties:
        endpoint: {type: string}
    record:
      identity: $.id
      fields:
        id: $.id
`
	mustNone(t, CheckManifest("providers/aws-s3.yaml", parse(t, doc)))
}

func TestCheckManifest_HoleInAnyOtherPositionMustResolveToAnInput(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /widgets/{widget_id}
    input:
      type: object
      properties: {}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeHoleIllegal)
	if p.Field != "operations.noop.http.path" {
		t.Errorf("Field = %q, want operations.noop.http.path", p.Field)
	}
}

func TestCheckManifest_PathGrammarRefusesDescentIndexAndWildcard(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{"recursive-descent", "$..id"},
		{"array-index", "$.items[0]"},
		{"wildcard", "$.items[*]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: "` + tc.path + `"
      fields:
        id: $.id
`
			got := CheckManifest("providers/broken.yaml", parse(t, doc))
			p := mustCode(t, got, schema.CodeMismatch)
			if p.Field != "operations.noop.record.identity" {
				t.Errorf("Field = %q, want operations.noop.record.identity", p.Field)
			}
		})
	}
}

func TestCheckManifest_PathGrammarAdmitsTheBracketForm(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        rate-limit: $["rate-limit"]
        nested: $.headers["x-request-id"]
`
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}

func TestCheckManifest_SeriesOperationOverAndIdentityBothRootAtDollar(t *testing.T) {
	doc := `kind: provider
provider: widgets
schema-version: 1
class: local
capabilities: [http]
operations:
  list:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /widgets
    record:
      over: $.body.result
      identity: $.id
      fields:
        id: $.id
        name: $.name
`
	mustNone(t, CheckManifest("providers/widgets.yaml", parse(t, doc)))
}

func TestCheckManifest_EnumerationsIsAMappingOfNameToBareScalars(t *testing.T) {
	doc := `kind: provider
provider: aws-s3
schema-version: 1
class: aws
capabilities: [http]
enumerations:
  region: {us-east-1: true}
operations:
  list:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "s3.{region}.amazonaws.com"
      path: /
`
	got := CheckManifest("providers/aws-s3.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "enumerations.region" {
		t.Errorf("Field = %q, want enumerations.region", p.Field)
	}
}

func TestCheckManifest_PatternsAdmitsExactlyTheThreeAndTheirClosedForms(t *testing.T) {
	got := CheckManifest("providers/broken.yaml", parse(t, patternsDoc(`throttle: {}`)))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "operations.list.patterns.throttle" {
		t.Errorf("Field = %q, want operations.list.patterns.throttle", p.Field)
	}
}

func TestCheckManifest_PaginationExactlyOneOfCursorOrPage(t *testing.T) {
	got := CheckManifest("providers/broken.yaml", parse(t, patternsDoc(`pagination: {}`)))
	mustCode(t, got, schema.CodeMismatch)

	doc := patternsDoc(`pagination: {cursor: {from: $.body.cursor, into: {query: cursor}}, page: {from: 1, into: {query: page}}}`)
	got = CheckManifest("providers/broken.yaml", parse(t, doc))
	mustCode(t, got, schema.CodeMismatch)
}

func TestCheckManifest_PaginationCursorIsClean(t *testing.T) {
	doc := patternsDoc(`pagination: {cursor: {from: $.body.cursor, into: {query: cursor}}}`)
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}

func TestCheckManifest_IntoNamesExactlyOneOfQueryOrHeader(t *testing.T) {
	doc := patternsDoc(`pagination: {page: {from: 1, into: {query: page, header: X}}}`)
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	mustCode(t, got, schema.CodeMismatch)
}

func TestCheckManifest_PollingIsClean(t *testing.T) {
	doc := patternsDoc(`polling: {interval: 5s, until: [{field: status, equals: running}]}`)
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}

// TestCheckManifest_PollingUntilFieldIsAPathWithoutRoot proves until:'s own
// root: a path in the grammar, written without the $ marker — status.code
// resolves and $status does not, a response having paths and no declared
// names (§12, issue #97).
func TestCheckManifest_PollingUntilFieldWithRootMarkerIsSchemaMismatch(t *testing.T) {
	doc := patternsDoc(`polling: {interval: 5s, until: [{field: $.status, equals: running}]}`)
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "operations.list.patterns.polling.until[0].field" {
		t.Errorf("Field = %q, want operations.list.patterns.polling.until[0].field", p.Field)
	}
}

// TestCheckManifest_PollingUntilOperandFaultIsTypeMismatch proves the
// operand-type rules apply to a polling Pattern's until: exactly as they do
// to a selector or a condition — the fault is authored and knowable offline
// wherever a predicate stands (§4, §5, §12, issue #97).
func TestCheckManifest_PollingUntilOperandFaultIsTypeMismatch(t *testing.T) {
	doc := patternsDoc(`polling: {interval: 5s, until: [{field: status, exists: false}]}`)
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	mustCode(t, got, CodePredicateTypeMismatch)
}

// TestCheckManifest_PollingUntilCarriesNoStep proves a polling Pattern's
// until: roots at the response object in hand rather than at an earlier
// Step's Record, so step: beside field: is unknown-key there and not the
// closed shape a condition's own field: reads under (§12, issue #97).
func TestCheckManifest_PollingUntilCarriesNoStep(t *testing.T) {
	doc := patternsDoc(`polling: {interval: 5s, until: [{field: status, step: whatever, equals: running}]}`)
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "operations.list.patterns.polling.until[0].step" {
		t.Errorf("Field = %q, want operations.list.patterns.polling.until[0].step", p.Field)
	}
}

func TestCheckManifest_RetryTakesOnlyAttempts(t *testing.T) {
	doc := patternsDoc(`retry: {attempts: 3, backoff: 5s}`)
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "operations.list.patterns.retry.backoff" {
		t.Errorf("Field = %q, want operations.list.patterns.retry.backoff", p.Field)
	}
}

// patternsDoc builds a minimal read Operation whose patterns: is exactly
// patterns, for the Patterns-focused tests above.
func patternsDoc(patterns string) string {
	return `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  list:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    patterns:
      ` + patterns + `
    record:
      over: $.body.result
      identity: $.id
      fields:
        id: $.id
`
}

func TestCheckManifest_AuthCarriesExactlyOneScheme(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
auth: {}
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	mustCode(t, got, schema.CodeMismatch)

	doc = `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
auth: {header: {name: X}, basic: {}}
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
`
	got = CheckManifest("providers/broken.yaml", parse(t, doc))
	mustCode(t, got, schema.CodeMismatch)
}

func TestCheckManifest_AuthBasicIsClean(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
auth: {basic: {}}
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        id: $.id
`
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}

func TestCheckManifest_SecretIsAListOfFieldsNames(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        id: $.id
        token: $.token
    secret: [token]
`
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))

	doc2 := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        id: $.id
    secret: {token: true}
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc2))
	mustCode(t, got, schema.CodeMismatch)
}

func TestCheckManifest_BodyTopLevelListOrScalarIsSchemaMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"list", `[1, 2, 3]`},
		{"scalar", `"just a string"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: mutate
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: /
      body: ` + tc.body + `
    record:
      identity: $.id
      fields:
        id: $.id
`
			got := CheckManifest("providers/broken.yaml", parse(t, doc))
			p := mustCode(t, got, schema.CodeMismatch)
			if p.Field != "operations.noop.http.body" {
				t.Errorf("Field = %q, want operations.noop.http.body", p.Field)
			}
		})
	}
}

func TestCheckManifest_BuiltinShellProviderHasNoNameOrKindMismatchExposure(t *testing.T) {
	// The built-in shell Provider authors its name outright, having no
	// file, so name-mismatch cannot reach it and there is no directory for
	// kind-mismatch to compare against (§3, §11) — CheckBuiltinShellProvider
	// runs neither check.
	got := CheckBuiltinShellProvider()
	mustNoCode(t, got, CodeKindMismatch)
	mustNoCode(t, got, CodeNameMismatch)
}

// TestCheckManifest_BuiltinShellProviderPassesEveryCheck is issue #92's own
// acceptance criterion: the built-in shell Provider passes every check this
// ticket adds, with no exemption of any kind (§3, §11).
func TestCheckManifest_BuiltinShellProviderPassesEveryCheck(t *testing.T) {
	mustNone(t, CheckBuiltinShellProvider())
}

// The Manifest's oracle (§4, issue #92): the checks that read a Manifest's
// own declarations against each other, with nothing but the file in hand.

// TestCheckManifest_CapabilityMismatchOverDeclared is a capability declared
// with no Operation naming it. Both halves of this check are written over http
// because the shell spelling of either is capability-reserved's site, and the
// mismatch row standing there is dropped (§11, §12) — which leaves a Manifest
// declaring the one unreserved member and using it nowhere as the whole of
// what over-declaring can now be.
func TestCheckManifest_CapabilityMismatchOverDeclared(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations: {}
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeCapabilityMismatch)
	if p.Field != "capabilities" {
		t.Errorf("Field = %q, want capabilities", p.Field)
	}
}

func TestCheckManifest_CapabilityMismatchUnderDeclared(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: []
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeCapabilityMismatch)
	if p.Field != "operations.noop.http" {
		t.Errorf("Field = %q, want operations.noop.http", p.Field)
	}
}

// §11's second rule about what an Extension may never be: it may never hold a
// Capability reserved to the Providers hyper ships (capability-reserved, §12,
// ADR-0004, ADR-0039). The subject is a Manifest loaded from providers/, which
// is why every case below runs through CheckManifest rather than through
// checkManifestBody, and why the built-in reaches none of them.

// TestCheckManifest_CapabilityReservedDeclaredSpelling is the first of the two
// spellings a Manifest reaches the Capability by: shell as a member of the
// top-level capabilities: list, cited at that member.
//
// It carries no Operation at all, so the only other row it could earn is the
// over-declared half of capability-mismatch — at that same member, for that
// same cause — and the assertion that exactly one problem stands is what says
// the row was dropped.
func TestCheckManifest_CapabilityReservedDeclaredSpelling(t *testing.T) {
	doc := `kind: provider
provider: local-shell
schema-version: 1
class: local
capabilities: [shell]
operations: {}
`
	got := CheckManifest("providers/local-shell.yaml", parse(t, doc))
	p := mustCode(t, got, CodeCapabilityReserved)
	if p.Field != "capabilities" || p.Line != 5 {
		t.Errorf("Field = %q, Line = %d, want capabilities at line 5", p.Field, p.Line)
	}
	if len(got) != 1 {
		t.Errorf("CheckManifest() = %+v, want the reserved row alone", got)
	}
}

// TestCheckManifest_CapabilityReservedDerivedSpelling is the second spelling,
// and the one the guarantee actually rests on: an Operation whose request
// block is shell: execs argv on the machine hyper runs on whatever the top
// level declares, so the row is cited at that key.
//
// The Manifest declares no Capability at all, so what stands at that same key
// without this check is the under-declared half of capability-mismatch — the
// row whose remedy is *declare it*, on a Manifest for which declaring it is
// the fault. Exactly one problem is what says that row was dropped.
func TestCheckManifest_CapabilityReservedDerivedSpelling(t *testing.T) {
	doc := `kind: provider
provider: local-shell
schema-version: 1
class: local
capabilities: []
operations:
  run:
    kind: read
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
`
	got := CheckManifest("providers/local-shell.yaml", parse(t, doc))
	p := mustCode(t, got, CodeCapabilityReserved)
	if p.Field != "operations.run.shell" || p.Line != 10 {
		t.Errorf("Field = %q, Line = %d, want operations.run.shell at line 10", p.Field, p.Line)
	}
	if len(got) != 1 {
		t.Errorf("CheckManifest() = %+v, want the reserved row alone", got)
	}
}

// TestCheckManifest_CapabilityReservedBothSpellings is the Manifest that
// satisfies the declared-equals-derived check exactly and is refused anyway:
// it earns a row at each site and no capability-mismatch, there being no
// disagreement for that check to find. It is the shape §13's guarantee is
// stated over — a Manifest in providers/ that declares shell, carries a
// shell: request block, satisfies declared-equals-derived exactly, and execs
// argv on the machine hyper runs on.
func TestCheckManifest_CapabilityReservedBothSpellings(t *testing.T) {
	doc := `kind: provider
provider: local-shell
schema-version: 1
class: local
capabilities: [shell]
operations:
  run:
    kind: read
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
`
	got := CheckManifest("providers/local-shell.yaml", parse(t, doc))
	mustNoCode(t, got, CodeCapabilityMismatch)
	if cited := fieldsOfCode(got, CodeCapabilityReserved); !slices.Equal(cited, []string{"capabilities", "operations.run.shell"}) {
		t.Errorf("cited %q, want a reserved row at each of the two sites", cited)
	}
}

// TestCheckManifest_CapabilityReservedDoesNotReachHTTP is the other half of
// the set: §12 reserves exactly one member, so a Manifest declaring and using
// http is untouched by this check whatever else it does.
func TestCheckManifest_CapabilityReservedDoesNotReachHTTP(t *testing.T) {
	mustNoCode(t, CheckManifest("providers/uptime.yaml", parse(t, uptime)), CodeCapabilityReserved)
	mustNoCode(t, CheckManifest("providers/cloudflare-dns.yaml", parse(t, cloudflareDNS)), CodeCapabilityReserved)
}

// TestCheckManifest_ARenamedForkOfTheBuiltinMayNotDeclareShell is §11's
// forkability limit read off one file: a built-in is forkable in form and not
// in power, so a copy of the built-in's own source in providers/ — renamed,
// which is what an Extension unable to shadow a built-in name must do — earns
// capability-reserved at its declared member and at each of its six
// Operations' shell: keys, and no name-mismatch, the rename being correct
// (§11, ADR-0039).
func TestCheckManifest_ARenamedForkOfTheBuiltinMayNotDeclareShell(t *testing.T) {
	doc := strings.Replace(BuiltinShellProviderYAML, "provider: shell\n", "provider: local-shell\n", 1)
	got := CheckManifest("providers/local-shell.yaml", parse(t, doc))
	mustNoCode(t, got, CodeNameMismatch)
	mustNoCode(t, got, CodeCapabilityMismatch)
	if n := countCode(got, CodeCapabilityReserved); n != 7 {
		t.Fatalf("CheckManifest() = %+v, want a reserved row at the declared member and at each of the six Operations", got)
	}
}

func TestCheckManifest_IdentityUndeclared(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeIdentityUndeclared)
	if p.Field != "operations.noop.record" {
		t.Errorf("Field = %q, want operations.noop.record", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentPaginationWithoutOver(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  list:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    patterns:
      pagination:
        cursor: {from: $.body.cursor, into: {query: cursor}}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.list.patterns.pagination" {
		t.Errorf("Field = %q, want operations.list.patterns.pagination", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentHostInputUnknownProperty(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: aws
capabilities: [http]
enumerations:
  region: [us-east-1, eu-central-1]
operations:
  list:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "s3.{region}.amazonaws.com"
      path: /
      host-input: endpoint
    input:
      type: object
      properties: {}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.list.http.host-input" {
		t.Errorf("Field = %q, want operations.list.http.host-input", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentHeadersEntryOwnedByAuthScheme(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
      headers: {authorization: "{token}"}
    input:
      type: object
      properties:
        token: {type: string}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.http.headers.authorization" {
		t.Errorf("Field = %q, want operations.noop.http.headers.authorization", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentShellOnlyWithAuth(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [shell]
auth:
  basic: {}
operations:
  run:
    kind: read
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "auth" {
		t.Errorf("Field = %q, want auth", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentHoleNamesObjectOrArrayInput(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: mutate
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: "/widgets/{tags}"
    input:
      type: object
      properties:
        tags: {type: array, items: {type: string}}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.http.path" {
		t.Errorf("Field = %q, want operations.noop.http.path", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentInputUnreached(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    input:
      type: object
      properties:
        unused: {type: string}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.input.properties.unused" {
		t.Errorf("Field = %q, want operations.noop.input.properties.unused", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentReadOrMutateWithoutRecord(t *testing.T) {
	for _, kind := range []string{"read", "mutate"} {
		t.Run(kind, func(t *testing.T) {
			doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: ` + kind + `
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
`
			got := CheckManifest("providers/broken.yaml", parse(t, doc))
			p := mustCode(t, got, CodeManifestInconsistent)
			if p.Field != "operations.noop" {
				t.Errorf("Field = %q, want operations.noop", p.Field)
			}
		})
	}
}

func TestCheckManifest_ManifestInconsistentDestroyWithRecord(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: destroy
    deadline: 1h
    http:
      method: DELETE
      host: "{from-target}"
      path: /widgets/{id}
    input:
      type: object
      properties:
        id: {type: string}
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.record" {
		t.Errorf("Field = %q, want operations.noop.record", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentSkipIfRecordedNotMutate(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    repeatability: skip-if-recorded
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: "/widgets/{id}"
    input:
      type: object
      properties:
        id: {type: string}
    record:
      identity: "{id}"
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.repeatability" {
		t.Errorf("Field = %q, want operations.noop.repeatability", p.Field)
	}
}

// TestCheckManifest_RunOnceIsNotAValueAnArtefactMayWrite is the fence the
// derived block earns by rendering the word: run-once is what an effectful
// Operation declaring no repeatability: *is*, and §12 gives it no keyword, so a
// Manifest attempting one is schema-mismatch like any other value outside an
// enum. Rendering a fact no artefact declares is exactly what `opaque` already
// does, and it stays that way only while writing it is refused (§3, §12).
func TestCheckManifest_RunOnceIsNotAValueAnArtefactMayWrite(t *testing.T) {
	got := CheckManifest("providers/broken.yaml", parse(t, `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  create_widget:
    kind: mutate
    repeatability: run-once
    deadline: 1h
    input:
      type: object
      properties:
        name: {type: string}
    http: {method: POST, host: "{from-target}", path: /widgets}
    record:
      identity: "{name}"
      fields: {id: $.body.id}
`))

	if p := mustCode(t, got, schema.CodeMismatch); p.Field != "operations.create_widget.repeatability" {
		t.Errorf("Field = %q, want operations.create_widget.repeatability", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentConcurrencyNotRead(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: mutate
    concurrency: 2
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: /widgets
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.concurrency" {
		t.Errorf("Field = %q, want operations.noop.concurrency", p.Field)
	}
}

func TestCheckManifest_ManifestInconsistentSkipIfRecordedIdentityFromResponse(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: /widgets
    record:
      identity: $.id
      fields:
        id: $.id
`
	got := CheckManifest("providers/broken.yaml", parse(t, doc))
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "operations.noop.record.identity" {
		t.Errorf("Field = %q, want operations.noop.record.identity", p.Field)
	}
}

// TestCheckManifest_SkipIfRecordedIdentityResolvingBeforeCallIsLegal is the
// acceptance criterion's other half: a template hole and $.command on a
// shell Operation both resolve before the call, so neither draws a code.
func TestCheckManifest_SkipIfRecordedIdentityResolvingBeforeCallIsLegal(t *testing.T) {
	t.Run("hole", func(t *testing.T) {
		doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
operations:
  create:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: /widgets
      body: {name: "{name}"}
    input:
      type: object
      properties:
        name: {type: string}
    record:
      identity: "{name}"
      fields:
        id: $.id
`
		mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
	})
	// The shell half runs checkManifestBody, a Manifest of that shape being
	// the built-in's and not a file in providers/ — see the note at the head
	// of this file. The claim is the same one either oracle would answer.
	t.Run("shell-command", func(t *testing.T) {
		doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [shell]
operations:
  run:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
`
		mustNone(t, checkManifestBody("providers/broken.yaml", parse(t, doc)))
	})
}

// reservedHeaderNames is the five headers §12 and §4 name — Host,
// Content-Length, Content-Type, Transfer-Encoding, Connection — each
// written here in a case that differs from its canonical spelling, proving
// header-reserved's comparison is case-insensitive across the whole set and
// not just the one name a single fixture happens to pick.
var reservedHeaderNames = []string{"host", "Content-Length", "content-type", "TRANSFER-ENCODING", "Connection"}

func TestCheckManifest_HeaderReserved(t *testing.T) {
	t.Run("auth scheme name", func(t *testing.T) {
		for _, name := range reservedHeaderNames {
			t.Run(name, func(t *testing.T) {
				doc := `kind: provider
provider: broken
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: ` + name + `}
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        id: $.id
`
				got := CheckManifest("providers/broken.yaml", parse(t, doc))
				p := mustCode(t, got, CodeHeaderReserved)
				if p.Field != "auth.header.name" {
					t.Errorf("Field = %q, want auth.header.name", p.Field)
				}
			})
		}
	})

	t.Run("headers entry", func(t *testing.T) {
		for _, name := range reservedHeaderNames {
			t.Run(name, func(t *testing.T) {
				doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
      headers: {` + name + `: example.com}
    record:
      identity: $.id
      fields:
        id: $.id
`
				got := CheckManifest("providers/broken.yaml", parse(t, doc))
				p := mustCode(t, got, CodeHeaderReserved)
				if p.Field != "operations.noop.http.headers."+name {
					t.Errorf("Field = %q, want operations.noop.http.headers.%s", p.Field, name)
				}
			})
		}
	})
}

func TestCheckManifest_ConcurrencyOneOnReadIsLegal(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  list:
    kind: read
    concurrency: 1
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    record:
      identity: $.id
      fields:
        id: $.id
`
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}

func TestCheckManifest_EmptyInputSchemaIsLegal(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  noop:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /
    input: {}
    record:
      identity: $.id
      fields:
        id: $.id
`
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}

func TestCheckManifest_DestroyWithNoRecordAndNoIdentityDrawsNoCode(t *testing.T) {
	doc := `kind: provider
provider: broken
schema-version: 1
class: local
capabilities: [http]
operations:
  delete:
    kind: destroy
    deadline: 1h
    http:
      method: DELETE
      host: "{from-target}"
      path: /widgets/{id}
    input:
      type: object
      properties:
        id: {type: string}
`
	mustNone(t, CheckManifest("providers/broken.yaml", parse(t, doc)))
}
