#!/usr/bin/env python3
"""
PROTOTYPE — hyper's MCP surface (issue #10).  THROWAWAY. Not production code.

Zero dependencies.  Two modes:

    python3 server.py --walkthrough    drive the scripted scenario, print every exchange
    python3 server.py                  speak MCP over stdio, for a real client

The data is fixed and fake.  What is being prototyped is the *shape* of the twelve tools,
the return envelope, and the Refusal path — not the engine behind them.

Scenario is #5's, deliberately: a procedure whose AI-raised destroy Bound of 5 meets a
selector that expands to 23.
"""

import json
import sys

PROTOCOL_VERSION = "2025-06-18"

# ---------------------------------------------------------------------------
# The return envelope (SURFACE.md §2)
# ---------------------------------------------------------------------------


def ret(text, rows, *, outcome=None, is_error=False, truncated=None, meta=None):
    """Every tool returns this shape. `text` leads with the outcome; `rows` is #5's renderer."""
    structured = {"ok": not is_error, "rows": rows}
    if outcome is not None:
        structured["outcome"] = outcome
    if meta is not None:
        structured["meta"] = meta
    structured["truncated"] = truncated
    return {
        "content": [{"type": "text", "text": text}],
        "structuredContent": structured,
        "isError": is_error,
    }


# ---------------------------------------------------------------------------
# Fixed fake world
# ---------------------------------------------------------------------------

PROVIDERS = [
    {"type": "provider", "name": "http", "origin": "builtin",
     "summary": "HTTP requests, TLS inspection, DNS.", "operation_count": 6,
     "digest": "sha256:9f2c...a71b"},
    {"type": "provider", "name": "proxmox", "origin": "extension",
     "summary": "Proxmox VE virtual machines and storage.", "operation_count": 14,
     "digest": "sha256:41de...0c93"},
    {"type": "provider", "name": "shell", "origin": "builtin",
     "summary": "Opaque local commands. Built-in only — never grantable to an Extension.",
     "operation_count": 1, "digest": "sha256:77aa...3e10"},
]

PROXMOX_OPS = [
    {"type": "operation", "name": "vm.list", "kind": "read", "opaque": False,
     "summary": "List virtual machines on a node."},
    {"type": "operation", "name": "vm.create", "kind": "mutate", "opaque": False,
     "summary": "Create a virtual machine from a template."},
    {"type": "operation", "name": "vm.resize_disk", "kind": "mutate", "opaque": False,
     "summary": "Resize a VM's primary disk."},
    {"type": "operation", "name": "vm.destroy", "kind": "destroy", "opaque": False,
     "summary": "Destroy a virtual machine and release its storage."},
]

HTTP_OPS = [
    {"type": "operation", "name": "get", "kind": "read", "opaque": False,
     "summary": "Issue an HTTP GET and record status, headers and timing."},
    {"type": "operation", "name": "tls.expiry", "kind": "read", "opaque": False,
     "summary": "Read a TLS certificate's not-after date."},
]

# The Manifest source for one Operation, returned verbatim (SURFACE.md §3).
# A Manifest is the same line-oriented format as a Definition — returning source is
# what teaches the agent the format it must author in.
VM_DESTROY_SOURCE = """\
operation vm.destroy
  kind        destroy
  summary     Destroy a virtual machine and release its storage.
  capability  http
  request
    method    DELETE
    path      /api2/json/nodes/{node}/qemu/{vmid}
  input
    node      string   required
    vmid      integer  required
  record
    cardinality  one
    identity     vmid
    kind         asset
  repeatability  repeatable
  deadline       120s
  concurrency    1
"""


def targets_rows():
    return [
        {"type": "target", "name": "local", "endpoint": "(this machine, public internet)",
         "accepts_kinds": ["read"], "grants_capabilities": ["http", "dns", "tls"],
         "credential_env": [], "credentials_present": True},
        {"type": "target", "name": "proxmox-staging", "endpoint": "https://pve-staging.internal:8006",
         "accepts_kinds": ["read", "mutate", "destroy"], "grants_capabilities": ["http", "tls-no-verify"],
         "credential_env": ["STAGING_PVE_TOKEN"], "credentials_present": True},
        {"type": "target", "name": "proxmox-prod", "endpoint": "https://pve.internal:8006",
         "accepts_kinds": ["read", "mutate", "destroy"], "grants_capabilities": ["http"],
         "credential_env": ["PROD_PVE_TOKEN"], "credentials_present": True},
    ]


RETIRE_SOURCE = [
    (1, "", "procedure retire-stale"),
    (2, "", "  cadence   0 3 * * 0        # weekly, Sunday 03:00 UTC — 4 runs/month"),
    (3, "", "  targets   proxmox-prod"),
    (4, "", ""),
    (5, "read", "  step list-stale"),
    (6, "", "    use       pve.inventory"),
    (7, "", "    operation vm.list"),
    (8, "", "    target    proxmox-prod"),
    (9, "", ""),
    (10, "DESTROY", "  step retire"),
    (11, "", "    use       pve.retirement"),
    (12, "", "    operation vm.destroy"),
    (13, "", "    target    proxmox-prod"),
    (14, "", '    select    assets where age > "90d"'),
    (15, "bound", "    bound     5"),
]


# ---------------------------------------------------------------------------
# The twelve tools
# ---------------------------------------------------------------------------


def t_providers():
    return ret("3 providers: 2 built-in, 1 extension.", PROVIDERS)


def t_provider(name):
    ops = {"proxmox": PROXMOX_OPS, "http": HTTP_OPS}.get(name, [])
    meta = {
        "proxmox": {"auth_scheme": "bearer-token", "capabilities_required": ["http"],
                    "digest": "sha256:41de...0c93", "schema_version": 1},
        "http": {"auth_scheme": "none", "capabilities_required": ["http", "dns", "tls"],
                 "digest": "sha256:9f2c...a71b", "schema_version": 1},
    }.get(name, {})
    # kind is on every row at LEVEL 2 — it answers #4's two-key check before any schema is fetched.
    return ret(f"{name}: {len(ops)} operations shown (kind on every row).", ops, meta=meta)


def t_operation(provider, name):
    if (provider, name) != ("proxmox", "vm.destroy"):
        return ret(f"{provider}.{name}: not stubbed in this prototype.", [])
    row = {
        "type": "operation_detail",
        "source": VM_DESTROY_SOURCE,          # the Manifest lines, verbatim
        "derived": {                           # only what is NOT in the source verbatim
            "capabilities": ["http"],          # derived; #7 requires exact equality with declared
            "bound_required": True,            # true iff kind == destroy
            "patterns_resolved": ["retry(3, backoff=exponential)"],
            "record_cardinality": "one",
            "record_identity": "vmid",
            "repeatability": "repeatable",
            "deadline_seconds": 120,
            "concurrency_limit": 1,
        },
    }
    return ret("proxmox.vm.destroy — destroy; a Bound is mandatory on any step using it.", [row])


def t_targets():
    return ret("3 targets; credentials present for all. Env var NAMES only, never values.",
               targets_rows())


def t_check():
    return ret("check: ok — 0 errors, 0 warnings across 4 artefacts.", [])


def t_review(artefact):
    gutter = [{"type": "gutter", "line": n, "marker": m, "source": s} for n, m, s in RETIRE_SOURCE]
    authority = [{
        "type": "authority",
        "definition": "pve.retirement",
        "declared_kinds": ["read", "destroy"],
        "target_accepts": ["read", "mutate", "destroy"],
        "intersection": ["read", "destroy"],
    }]
    # Every flag cites a line the gutter already marked, and introduces no claim of its own (#5).
    flags = [
        {"type": "flag", "code": "DESTROY", "cites_line": 10,
         "text": "step retire is destroy against proxmox-prod"},
        {"type": "flag", "code": "WIDENED", "cites_line": 15,
         "text": "bound 3 -> 5 since definition revision a91f0c2"},
        {"type": "flag", "code": "CADENCE", "cites_line": 2,
         "text": "0 3 * * 0 -> 4 runs/month on a destroy procedure"},
    ]
    rendered = ["  REVIEW  " + artefact, ""]
    for n, m, s in RETIRE_SOURCE:
        rendered.append(f"  {m:>8} │ {n:>3} │ {s}")
    rendered += ["", "  AUTHORITY  pve.retirement  declared [read destroy]",
                 "             proxmox-prod    accepts  [read mutate destroy]",
                 "             ------------------------------------------------",
                 "             intersection    [read destroy]", ""]
    rendered.append("  FLAGS")
    for f in flags:
        rendered.append(f"    {f['code']:<8} line {f['cites_line']:>3}  {f['text']}")
    return ret("\n".join(rendered), gutter + authority + flags)


def t_probe(provider, operation, inputs):
    # read Kind only, local Target only, no Record, no Journal entry.  A probe is NOT a Run:
    # no Trigger, no Provenance, no Disposition, no outcome triple.
    row = {"type": "observation", "provider": provider, "operation": operation,
           "input": inputs, "status": 200, "elapsed_ms": 143,
           "recorded": False, "journal_entry": None}
    return ret("probe: 200 in 143ms. Not a Run — no Record, no Journal entry, no Provenance.",
               [row])


def t_run(procedure, target, dry_run=False, secret_sink=None):
    """The Refusal path — #5's scenario. Bound 5 meets an Expansion of 23."""
    steps = [
        {"type": "step", "index": 1, "disposition": "ran", "definition": "pve.inventory",
         "operation": "vm.list", "target": target, "kind": "read"},
        {"type": "step", "index": 2, "disposition": "refused", "definition": "pve.retirement",
         "operation": "vm.destroy", "target": target, "kind": "destroy"},
    ]
    refusal = {"type": "refusal", "check": "bound-exceeded", "step": 2, "target": target,
               "declared": 5, "observed": 23,
               "artefact": "procedures/retire-stale.hyper", "line": 15}
    edits = [
        {"type": "edit_option", "file": "procedures/retire-stale.hyper", "line": 15,
         "field": "bound", "from": "5", "to": "23", "effect": "permits all 23"},
        {"type": "edit_option", "file": "procedures/retire-stale.hyper", "line": 14,
         "field": "select", "from": 'age > "90d"', "to": 'age > "365d"',
         "effect": "expands to 4"},
    ]
    text = f"""\
REFUSED — bound exceeded. No effect reached the world at step 2.

  procedures/retire-stale.hyper
   14 │     select    assets where age > "90d"
   15 │     bound     5
                      ^
                      expansion produced 23 records; bound is 5

  Step 1 completed and its 31 records are written and durable.
  Step 2 wrote nothing.

  EDIT ONE OF
    line 15  bound   5           -> 23           permits all 23
    line 14  select  age > "90d" -> age > "365d" expands to 4

  There is no bypass. Re-running this call unchanged will refuse identically —
  the only way past this is an edit to procedures/retire-stale.hyper and a re-review."""
    return ret(text, steps + [refusal] + edits, outcome="refused", is_error=True)


def t_runs(limit=200):
    rows = [
        {"type": "run", "id": "r-0f31c8", "started": "2026-08-02T03:00:04Z", "trigger": "cadence/actions",
         "outcome": "refused", "procedure": "retire-stale", "targets": ["proxmox-prod"],
         "hyper_version": "0.4.1"},
        {"type": "run", "id": "r-0e77a2", "started": "2026-07-26T03:00:03Z", "trigger": "cadence/actions",
         "outcome": "completed", "procedure": "retire-stale", "targets": ["proxmox-prod"],
         "hyper_version": "0.4.0"},
        {"type": "run", "id": "r-0e10bd", "started": "2026-07-24T11:42:55Z", "trigger": "manual/laptop",
         "outcome": "failed", "procedure": "retire-stale", "targets": ["proxmox-prod"],
         "hyper_version": "0.4.0"},
    ]
    trunc = {"axis": "time", "returned": len(rows), "dropped": 2840,
             "hint": "narrow with `since` or `procedure`; 2840 runs fall outside the returned window"}
    return ret(f"{len(rows)} runs shown, 2840 dropped — narrow the query, there is no page two.",
               rows, truncated=trunc)


def t_run_show(run_id):
    rows = [
        {"type": "disposition", "step": 1, "state": "ran"},
        {"type": "disposition", "step": 2, "state": "refused"},
        {"type": "disposition", "step": 3, "state": "never-reached"},
        {"type": "provenance", "definition_revision": "4d7e118", "manifest_digest": "sha256:41de...0c93",
         "extension_digest": "sha256:41de...0c93", "repo_revision": "1bb9e47", "hyper_version": "0.4.1"},
        {"type": "expansion", "step": 2, "selector": 'assets where age > "90d"',
         "expanded_to": 23, "bound": 5},
    ]
    return ret(f"{run_id}: refused at step 2. 1 ran, 1 refused, 1 never reached.", rows)


def t_diff(since):
    rows = [
        # YOU DID THIS
        {"type": "asset", "change": "tomb", "key": "proxmox-prod/pve.retirement/vmid=812",
         "version": 4, "note": "confirmed destroyed"},
        {"type": "asset", "change": "new", "key": "proxmox-prod/pve.provision/vmid=901", "version": 1},
        # THE WORLD MOVED
        {"type": "observation", "change": "mut", "key": "proxmox-prod/pve.inventory/vm-count",
         "from": 34, "to": 31, "version": 88},
        # THE CODE MOVED — provenance drift as a first-class diff event (#5 decision 4)
        {"type": "code", "definition_revision_from": "a91f0c2", "definition_revision_to": "4d7e118",
         "summary": "pve.retirement: step retire, bound 3 -> 5"},
    ]
    return ret(f"since {since}: 2 asset changes, 1 observation change, 1 code change.", rows)


def t_records(target=None, definition=None, name=None, history=False):
    rows = [
        {"type": "record", "key": {"target": "proxmox-prod", "definition": "pve.provision",
                                   "name": "vmid=901"},
         "version": 1, "record_kind": "asset", "tombstoned": False,
         "secret_fields": ["root_password"],   # presence-only marker, per #9 — never a digest
         "provenance": {"definition_revision": "4d7e118", "hyper_version": "0.4.1"}},
    ]
    return ret("1 record. secret_fields is presence-only — no value, no digest.", rows)


# ---------------------------------------------------------------------------
# Tool registry — twelve.  `install` is deliberately absent (SURFACE.md §7).
# ---------------------------------------------------------------------------

TOOLS = [
    ("providers", "List every Provider. Level 1 of 3.", {}, t_providers),
    ("provider", "One Provider's Operations, with Kind on every row. Level 2 of 3.",
     {"name": "string"}, lambda a: t_provider(a["name"])),
    ("operation", "One Operation: Manifest source plus derived facts. Level 3 of 3.",
     {"provider": "string", "name": "string"}, lambda a: t_operation(a["provider"], a["name"])),
    ("targets", "Target declarations: accepted Kinds, granted Capabilities, credential env NAMES.",
     {}, t_targets),
    ("check", "Static oracle. Offline, no credentials. Errors positioned by file and line.",
     {"paths": "array?"}, lambda a: t_check()),
    ("review", "Render an artefact's review surface: gutter, AUTHORITY, FLAGS.",
     {"artefact": "string"}, lambda a: t_review(a["artefact"])),
    ("run", "Execute a Procedure. No inputs — a Procedure is fully bound by its artefact.",
     {"procedure": "string", "target": "string", "dry_run": "boolean?", "secret_sink": "string?"},
     lambda a: t_run(a["procedure"], a["target"], a.get("dry_run", False), a.get("secret_sink"))),
    ("probe", "One-off read. read Kind only, local Target only. Not a Run.",
     {"provider": "string", "operation": "string", "inputs": "object"},
     lambda a: t_probe(a["provider"], a["operation"], a["inputs"])),
    ("runs", "Journal entries. Truncates with a marker; there is no cursor.",
     {"since": "string?", "procedure": "string?", "outcome": "string?", "limit": "integer?"},
     lambda a: t_runs(a.get("limit", 200))),
    ("run_show", "One Run: Dispositions, Provenance, Expansions.",
     {"id": "string"}, lambda a: t_run_show(a["id"])),
    ("diff", "The three tables: YOU DID THIS / THE WORLD MOVED / THE CODE MOVED.",
     {"since": "string?", "between": "array?", "target": "string?", "limit": "integer?"},
     lambda a: t_diff(a.get("since", "7d"))),
    ("records", "Record versions with Provenance and presence-only secret markers.",
     {"target": "string?", "definition": "string?", "name": "string?", "history": "boolean?"},
     lambda a: t_records(**{k: v for k, v in a.items() if k in
                            ("target", "definition", "name", "history")})),
]

DISPATCH = {name: fn for name, _, _, fn in TOOLS}


def tool_list():
    out = []
    for name, desc, params, _ in TOOLS:
        props, required = {}, []
        for pname, ptype in params.items():
            optional = ptype.endswith("?")
            props[pname] = {"type": ptype.rstrip("?")}
            if not optional:
                required.append(pname)
        out.append({"name": name, "description": desc,
                    "inputSchema": {"type": "object", "properties": props, "required": required}})
    return out


def call(name, args):
    fn = DISPATCH.get(name)
    if fn is None:
        # Unknown tool is a PROTOCOL error, never a domain outcome (SURFACE.md §4).
        raise KeyError(name)
    return fn(args) if fn.__code__.co_argcount else fn()


# ---------------------------------------------------------------------------
# Walkthrough
# ---------------------------------------------------------------------------

SCRIPT = [
    ("The three questions, in order — which Provider?", "providers", {}),
    ("...which Operation? (Kind is here, at level 2)", "provider", {"name": "proxmox"}),
    ("...how do I call it? (source + derived)", "operation",
     {"provider": "proxmox", "name": "vm.destroy"}),
    ("What may I do, and against what?", "targets", {}),
    ("The one-off read. Not a Run.", "probe",
     {"provider": "http", "operation": "get", "inputs": {"url": "https://example.com"}}),
    ("The static oracle — offline, no credentials.", "check", {}),
    ("What a human is asked to approve.", "review", {"artefact": "procedures/retire-stale.hyper"}),
    ("Execute. Bound 5 meets an Expansion of 23.", "run",
     {"procedure": "retire-stale", "target": "proxmox-prod"}),
    ("The same call again, verbatim — the loop-breaker.", "run",
     {"procedure": "retire-stale", "target": "proxmox-prod"}),
    ("Truncation with no cursor.", "runs", {}),
    ("One Run's Dispositions.", "run_show", {"id": "r-0f31c8"}),
    ("The three tables.", "diff", {"since": "2026-07-25"}),
    ("Records, with presence-only secret markers.", "records", {"target": "proxmox-prod"}),
]


def walkthrough():
    w = 78
    print("=" * w)
    print("  PROTOTYPE — hyper's MCP surface (#10).  Fake data; the shape is the point.")
    print("=" * w)
    print(f"\n  {len(TOOLS)} tools exposed:  " + ", ".join(n for n, _, _, _ in TOOLS))
    print("  Deliberately absent: install  (acquisition, not derivation — SURFACE.md §7)\n")

    for i, (note, name, args) in enumerate(SCRIPT, 1):
        print("-" * w)
        print(f"  [{i:>2}] {note}")
        print(f"       → {name}({json.dumps(args)[1:-1] or ''})")
        print("-" * w)
        res = call(name, args)
        flag = "  isError: true" if res["isError"] else ""
        print(f"\n  text:{flag}")
        for line in res["content"][0]["text"].splitlines():
            print(f"    {line}")
        sc = res["structuredContent"]
        print(f"\n  structuredContent: ok={sc['ok']}"
              + (f"  outcome={sc['outcome']}" if "outcome" in sc else "")
              + f"  rows={len(sc['rows'])}")
        by_type = {}
        for r in sc["rows"]:
            by_type.setdefault(r["type"], []).append(r)
        for rtype, rs in by_type.items():
            print(f"    {rtype} × {len(rs)}")
            print(f"      {json.dumps(rs[0], separators=(',', ':'))[:150]}")
        if sc.get("truncated"):
            print(f"    truncated: {json.dumps(sc['truncated'], separators=(',', ':'))}")
        print()

    print("=" * w)
    print("  Try a tool that does not exist — a PROTOCOL error, never a domain outcome:")
    try:
        call("install", {"name": "proxmox"})
    except KeyError as e:
        print(f"    JSON-RPC -32602: unknown tool {e}")
    print("=" * w)


# ---------------------------------------------------------------------------
# stdio MCP
# ---------------------------------------------------------------------------


def serve():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        req = json.loads(line)
        method, rid = req.get("method"), req.get("id")
        if method == "initialize":
            result = {"protocolVersion": PROTOCOL_VERSION,
                      "capabilities": {"tools": {}},
                      "serverInfo": {"name": "hyper-prototype", "version": "0.0.0"}}
        elif method == "tools/list":
            result = {"tools": tool_list()}
        elif method == "tools/call":
            p = req["params"]
            try:
                result = call(p["name"], p.get("arguments", {}))
            except KeyError:
                err = {"jsonrpc": "2.0", "id": rid,
                       "error": {"code": -32602, "message": f"unknown tool: {p['name']}"}}
                print(json.dumps(err), flush=True)
                continue
        elif rid is None:
            continue  # notification
        else:
            result = {}
        print(json.dumps({"jsonrpc": "2.0", "id": rid, "result": result}), flush=True)


if __name__ == "__main__":
    walkthrough() if "--walkthrough" in sys.argv else serve()
