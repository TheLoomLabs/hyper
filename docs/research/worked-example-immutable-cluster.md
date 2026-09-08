# A worked scenario: an immutably provisioned Kubernetes cluster

**Not yet frozen.** This is part one of a conformance exercise, and part one authors: it states the
scenario, gives the reason for it, and holds the artefacts in full. The Run, the Store files, the
renderings and the findings are part two, and so is the freeze. Until part two lands, this file may be
edited. When it is frozen it will name one `hyper` commit and never be updated again, on the standing
rule [the Tailscale exercise](worked-example-tailscale-ci-keys.md) states: if the spec has moved on,
re-work the scenario from scratch rather than editing the file.

Worked against `hyper` **0.0.3-alpha**, built from this tree at commit `e599c35`. Everything derivable
is derived — the Manifest digests, the git blob ids and the `check` counts below were computed from the
actual files rather than invented, so the numbers agree with each other and can be re-checked.

## Method, and what this is evidence of

This is not research against an external source. It is the specification read as an implementer would
read it — [`docs/spec/`](../spec/) §0–§13, [`CONTEXT.md`](../../CONTEXT.md) and the
[ADRs](../adr/) — and then *used*, once, on a scenario nobody had written down. The scenario was chosen
to make several rules meet rather than to exercise any one of them.

What part one is evidence of is one claim and no more: **the artefacts an immutably provisioned cluster
needs are authorable against the shipped product, with no new Capability, no new artefact key and no
code.** The claim is driven rather than reasoned. The eight files in §1 were written into a scratch
repository and `hyper check` was run against a binary built from this tree; every rule the prose below
says bit was made to bite, by breaking the artefact in one place and reading the code back. The
negative controls are in §1's last subsection for that reason: without them *`check` accepts this* and
*`check` never looks here* are the same observation.

What it is **not** evidence of is that the Run composes, or that the Comparison renders anything worth
reading. That is part two's, and part two's findings section is the deliverable of the whole exercise —
here, as in the Tailscale one, everything above the findings is a demonstration and the findings are the
result. Nothing here is normative: the specification is `docs/spec/`, the rationale is `docs/adr/`.

The scratch repository is not part of this tree, and nothing here depends on the copy that produced
these numbers: it is reproducible from §1 alone, with `git hash-object` and `sha256sum`, which is where
every id below came from.

---

## 0. The scenario, and why this one

A small team runs one Kubernetes cluster on Google Compute Engine and provisions it **immutably**:
every machine is created already configured, and no Step ever reaches inside a running one. There is no
configuration management, no agent, no `ssh` anywhere in the repository, and nothing to converge —
a machine that is wrong is replaced rather than repaired.

The configuration travels in the create request's instance metadata, as the `user-data` a machine's
cloud-init reads. That value is a scalar in a request body, which makes it a **reviewed artefact**: the
whole of what the machine will do to itself is on the screen beside the Step that creates it, in the
gutter-annotated surface a human approves (§8).

One `hyper` Run forms the cluster:

1. **the control plane** is created with a reserved external address and a cloud-init that runs
   `kubeadm init`, then publishes the join command to Secret Manager;
2. **three workers** are created from one Step over a list of node names, each with a cloud-init that
   fetches that join command from Secret Manager and runs it;
3. **the cluster's own API is read**, which is the **paired `read`**: a `read` Step written beside an
   effectful one for no reason but to record what the world says afterwards. It is a practice an
   author performs rather than a thing the model has — `CONTEXT.md` is a glossary and not a pattern
   book, so it gets no term — and it is the whole of what stands between a Comparison that renders an
   Observation and one that renders *a call was accepted* and stops.

The join secret goes **machine-to-machine**. A GCE instance authenticates to Secret Manager with the
service-account identity it already carries, so the control plane publishes and the workers fetch
without any credential being written anywhere: the token crosses no artefact, no Record and no Step's
arguments. `hyper` holds the instruction to fetch a secret and never the secret — which is
[ADR-0007](../adr/0007-hyper-never-stores-a-secret.md) holding at a place it was not written for.

**The shape was chosen because these rules only bite when they meet:**

- a Definition that **effects and does not observe** ([ADR-0032](../adr/0032-a-definition-observes-or-effects-never-both.md)),
  claiming `mutate` against a locally authored `http` Manifest — so reading the result back is a
  *second* Definition, against a second Provider and a second Target;
- an **Expansion over the node list**, so one Step calls for three members, each of them one Record
  series and one `skip-if-recorded` test ([ADR-0056](../adr/0056-skip-if-recorded-tests-a-record-not-a-step.md));
- the **paired `read`** beside the effectful Steps, which is the practice that gives the Comparison
  something about the world to render;
- `user-data` as a **multi-line value**, which the strict YAML subset admits as a block scalar
  ([ADR-0023](../adr/0023-the-authoring-format-is-a-strict-yaml-subset.md)) and which nothing in the
  format shortens, quotes or re-reads;
- a **Target declaration granting `http` and enumerating its hosts**, with no wildcard available
  ([ADR-0024](../adr/0024-local-grants-enumerated-hosts-like-any-other-target.md),
  [ADR-0029](../adr/0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md)) — a grant is a
  list of hosts, and a glob in one is a host nothing matches rather than a pattern something does.

### Two rules shaped the artefacts before anything ran

**A grant enumerates hosts, and a host has no port.** `hosts:` is a list of hosts and the scheme is
`https` with no second one ([ADR-0082](../adr/0082-the-scheme-is-https-and-there-is-no-second-one.md)),
so there is no position in any of the five artefacts where `:6443` belongs. The control-plane endpoint
therefore has to answer on 443 — which is why the `kubeadm` configuration in the Step below sets
`bindPort: 443` and a `controlPlaneEndpoint` on the same port. The constraint arrived from `hyper` and
was paid for in the cluster's own configuration, which is the direction this exercise is looking for.

**An API with a genuinely optional parameter is two Operations** (§3, §13). Every input an Operation
declares is supplied — there is no `null` and no key-omission syntax — so the reserved external address
the control plane takes and the workers do not cannot be one input that is sometimes absent. `gce`
therefore declares `create_control_plane` and `create_worker`, which differ in exactly that one nested
body key and in the `role` label beside it. The cost §13 names is real and is paid here in full: two
Operations, two input schemas, and one projection written twice.

### What the scenario needs that `hyper` does not supply

Stated as costs rather than left to be discovered.

- **A secret store**, and it is not `hyper`'s. Nothing here forms a cluster on its own: without Secret
  Manager the join command has nowhere to go that is not an artefact, a Record or a Step's argument.
- **A machine image built out of band.** `containerd`, `kubelet`, `kubeadm` and the Google Cloud CLI
  are in `projects/hyper-example/global/images/family/k8s-node-1-33`, not in the cloud-init: `hyper`
  has no `file` Capability and never places bytes on a machine
  ([ADR-0157](../adr/0157-a-capability-is-an-effect-hyper-performs-with-a-credential-it-already-has-a-position-for.md)),
  so an image is where the parts too large for a metadata value live.
- **A DNS name and a reserved address**, both standing before the Run. `k8s.hyper-example.dev` is what
  the Target grants and what `kubeadm` is told the control plane is, and neither can be learned from a
  machine this Run is about to create.
- **A short-lived OAuth access token in the environment.** An Auth scheme decorates a request and never
  performs one ([ADR-0031](../adr/0031-an-auth-scheme-is-a-header-and-a-placement-never-a-protocol.md)),
  so nothing in `hyper` exchanges a service-account key for a bearer token: `GOOGLE_OAUTH_ACCESS_TOKEN`
  is resolved at Run start from an environment the operator built, in the
  `op run --` / `aws-vault exec --` shape §3 already names.
- **The cluster's own certificate authority, reached through the environment.** The control plane signs
  its own API server certificate, so the paired `read` verifies against a root no public store holds.
  The route is `SSL_CERT_FILE`, which Go reads and `internal/capability` cannot override — trust is
  unreachable from where the agent writes, which is
  [ADR-0105](../adr/0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)'s
  decisive property. Part two exercises that route and states its honest limit.

Nothing on that list is a gap this exercise found. Each is a boundary the corpus already draws, met
from the far side.

---

## 1. The artefacts

**Eight files, five kinds.** Two Manifests, two Target declarations, two Definitions, one Procedure and
one Repository declaration. The paired `read` is what doubles the first three kinds: a Definition
observes or it effects and never both, so reading a cluster back is a second Definition, against a
second Provider, bound to a second Target. **`shell` is granted nowhere in any of them**, which is
what *no Step reaches inside a running machine* means as a fact `check` can hold rather than as a
promise the prose makes.

### `providers/gce.yaml`

```yaml
kind: provider
provider: gce
schema-version: 1
class: google-cloud
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  create_control_plane:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 120s
    patterns:
      retry: {attempts: 3}
    http:
      method: POST
      host: "{from-target}"
      path: /compute/v1/projects/{project}/zones/{zone}/instances
      body:
        name: "{name}"
        machineType: "{machine_type}"
        canIpForward: true
        disks:
          - boot: true
            autoDelete: true
            initializeParams:
              sourceImage: "{image}"
              diskSizeGb: "{disk_gb}"
        networkInterfaces:
          - network: "{network}"
            accessConfigs:
              - name: external
                type: ONE_TO_ONE_NAT
                natIP: "{static_ip}"
        serviceAccounts:
          - email: "{service_account}"
            scopes: ["https://www.googleapis.com/auth/cloud-platform"]
        labels:
          cluster: "{cluster}"
          role: control-plane
        metadata:
          items:
            - key: user-data
              value: "{user_data}"
    input:
      type: object
      properties:
        project: {type: string}
        zone: {type: string}
        name: {type: string}
        machine_type: {type: string}
        image: {type: string}
        disk_gb: {type: string}
        network: {type: string}
        static_ip: {type: string}
        service_account: {type: string}
        cluster: {type: string}
        user_data: {type: string}
    record:
      identity: "{name}"
      fields:
        operation: $.body.name
        status: $.body.status
        target_link: $.body.targetLink
        started: $.body.startTime
  create_worker:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 120s
    patterns:
      retry: {attempts: 3}
    http:
      method: POST
      host: "{from-target}"
      path: /compute/v1/projects/{project}/zones/{zone}/instances
      body:
        name: "{name}"
        machineType: "{machine_type}"
        canIpForward: true
        disks:
          - boot: true
            autoDelete: true
            initializeParams:
              sourceImage: "{image}"
              diskSizeGb: "{disk_gb}"
        networkInterfaces:
          - network: "{network}"
        serviceAccounts:
          - email: "{service_account}"
            scopes: ["https://www.googleapis.com/auth/cloud-platform"]
        labels:
          cluster: "{cluster}"
          role: worker
        metadata:
          items:
            - key: user-data
              value: "{user_data}"
    input:
      type: object
      properties:
        project: {type: string}
        zone: {type: string}
        name: {type: string}
        machine_type: {type: string}
        image: {type: string}
        disk_gb: {type: string}
        network: {type: string}
        service_account: {type: string}
        cluster: {type: string}
        user_data: {type: string}
    record:
      identity: "{name}"
      fields:
        operation: $.body.name
        status: $.body.status
        target_link: $.body.targetLink
        started: $.body.startTime
```

Two Operations for one API call, on the rule §3 states and §13 costs: `create_control_plane` carries an
`accessConfigs:` entry with a `natIP` and `create_worker` does not, and an input that is sometimes
absent is not writable. Everything else about them is one shape written twice, which is what the cost
looks like from inside.

`identity: "{name}"` is a **template hole and not a response path**, which `skip-if-recorded` requires:
the test reads the head of the series the call would write under, so the name has to exist before the
call it is deciding on (§3, §6, ADR-0056). It is also the only identity available: GCE answers an
`instances.insert` with a *zone operation* — its own long-running-job object, which shares a word with
`hyper`'s Operation and is not one — whose `name` and `id` belong to the job and not to the instance.
The response path that looks like the obvious identity therefore names the wrong thing, and is refused
where it is written before it can.

**What the projection records is the call, not the machine.** All four fields come off that job object,
which is why none of them is named after a server: the Asset says *an insert was accepted under this
name, and here is the job that carries it*. It says nothing about whether a machine booted, whether
cloud-init succeeded, or whether `kubeadm` formed anything. That is not an omission in the Manifest —
it is the whole of what `hyper`'s own effect reached, and it is why the paired `read` below is not
decoration.

`canIpForward: true` is spelled a boolean and reaches the wire as one; `diskSizeGb: "{disk_gb}"` names
an input declared `{type: string}` and reaches it as a JSON string, which is GCE's int64-as-string and
not a quoting accident. A hole carries the declared type of the input it resolves to, and a literal
carries its own YAML 1.2 core type — the two halves of
[ADR-0078](../adr/0078-a-body-literal-is-typed-by-its-spelling-and-a-hole-by-its-input.md), a few lines
apart in one file.

`value: "{user_data}"` sits under `metadata.items`, which is a list of mappings: a hole fills a value
position anywhere in the tree, a list member included, and never a mapping key. `concurrency:` is
declared nowhere — an effectful Expansion runs serially and a limit on one is `manifest-inconsistent`
([ADR-0045](../adr/0045-a-concurrency-limit-is-a-reads-and-is-one-unless-declared.md)). `origin:` is
absent, so this is a locally authored Provider and `origin_digest` will be absent from every Provenance
part two writes.

Digest over these exact bytes:
`sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84`

### `providers/kubernetes.yaml`

```yaml
kind: provider
provider: kubernetes
schema-version: 1
class: kubernetes
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  list_nodes:
    kind: read
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /api/v1/nodes
    record:
      over: $.body.items
      identity: $.metadata.name
      fields:
        name: $.metadata.name
        created: $.metadata.creationTimestamp
        kubelet_version: $.status.nodeInfo.kubeletVersion
        os_image: $.status.nodeInfo.osImage
        unschedulable: $.spec.unschedulable
```

The smallest artefact that makes the exercise worth doing. One `read`, no `input:` at all — an
Operation that takes none, so the Step that binds it writes no `args:` — and a `series` projection over
the node list.

`repeatability:` is omitted because a `read` has one legal value (§12), and `concurrency:` because this
Operation is invoked once and no number has been measured. The Auth scheme is the same `header:` the
cloud Manifest uses, and the credential it carries is a Kubernetes ServiceAccount token: an Auth scheme
is a header and a placement, so one scheme serves two entirely unrelated systems without either
Manifest knowing anything about the other.

**`ready` is not among the fields, and it could not be.** A node's readiness lives in
`status.conditions`, a list whose members are told apart by a `type` key, and the path grammar is `$`,
`.member` and `["member"]` with no indexing and no iteration (§12) — the identity-is-array-position
hazard a declared Record identity exists to close. What is projected instead is what the collection
member carries flat: the node's name, when the API server first saw it, the kubelet it is running, and
`unschedulable`, whose **absence** is the ordinary state and is a predicate fact rather than a type
(§3, §12).

That is enough for the pairing to do its work. An Observation under this Definition says *the API
server knows a node by this name, running this kubelet, since this instant* — which is a fact about the
cluster that no Asset written by the Steps above could contain.

Digest over these exact bytes:
`sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6`

### `targets/gce-prod.yaml`

```yaml
kind: target-declaration
target: gce-prod
class: google-cloud
kinds: [mutate]
capabilities: [http]
hosts: [compute.googleapis.com]
auth:
  token: {env: GOOGLE_OAUTH_ACCESS_TOKEN}
```

`hosts:` is present exactly because `capabilities:` grants `http`; either without the other is
`target-inconsistent` (§4). The grant holds one host, so the candidate set `{from-target}` expands to
intersects with that grant to exactly one host, and `hyper` fills it — no `host-input:` anywhere in the
Manifest
([ADR-0029](../adr/0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md)).

`kinds: [mutate]` and nothing else. This Target accepts no `read` because nothing in this repository
reads the cloud API: what the operator wants to know about the cluster is what the cluster says, not
what the cloud's own inventory says. A Target accepts, a Definition claims, and a Step runs in the
intersection (§4) — so narrowing here is a narrowing a reviewer can see in one line.

No `opaque-destroy:`, nothing bound here being opaque, and no `withhold:` — the invoking environment
carries a bearer token this Run resolves and nothing a child would inherit, there being no child.

### `targets/cluster-api.yaml`

```yaml
kind: target-declaration
target: cluster-api
class: kubernetes
kinds: [read]
capabilities: [http]
hosts: [k8s.hyper-example.dev]
auth:
  token: {env: CLUSTER_OBSERVER_TOKEN}
```

One host, `read` alone, and a credential slot of its own. The two Targets are two credentials and
therefore two Targets: a Target is the unit of blast radius and of credentials at once, and a
declaration cannot hold two different secrets for one scheme (§3).

`k8s.hyper-example.dev` carries no port, for the reason §0 states, and no scheme, because there is
only one. What the Target does *not* carry is the root that verifies it — a PEM in a reviewed artefact
would be a blob a reviewer cannot check, and `SSL_CERT_FILE` is where trust lives instead (ADR-0105).

### `definitions/cluster-machines.yaml`

```yaml
kind: definition
definition: cluster-machines
provider: gce
kinds: [mutate]
targets: [gce-prod]
```

`kinds: [mutate]` with no `destroy:` beside it: this Procedure creates machines and ends none. Deleting
a node would be a `destroy` Operation named here by name, granularity following severity (§3, §4), and
this repository does not have one — the immutable model replaces a cluster by forming another and
pointing the endpoint at it, which is a decision the operator makes and not a Step.

No argument value lives here. The Definition claims authority and the values are the Step's, because a
value living on a Definition would force a second, `AUTHORITY`-shaped table just to re-show it — the
gutter renders what reaches the world from the file being read
([ADR-0026](../adr/0026-the-gutter-annotates-a-table-aggregates-and-one-surface-editorialises.md)).

### `definitions/cluster-membership.yaml`

```yaml
kind: definition
definition: cluster-membership
provider: kubernetes
kinds: [read]
targets: [cluster-api]
```

The paired `read`'s half of the pair, and the reason it is a separate file. A Definition observes or it
effects, so `read` may not stand in `kinds:` beside `mutate`
([ADR-0032](../adr/0032-a-definition-observes-or-effects-never-both.md)); the Definition is also the
segment of a Record's identity that keeps two series apart, so the Observations this one writes and the
Assets `cluster-machines` owns can never meet. `cp-1` the Asset and `cp-1` the Observation are two
Records with one name and no relation `hyper` asserts between them — which is the honest reading, since
one is *a create call was accepted* and the other is *the API server knows this node*.

### `procedures/form-cluster.yaml`

```yaml
kind: procedure
procedure: form-cluster
targets: [gce-prod, cluster-api]
steps:
  # the control plane is created already configured: kubeadm runs from cloud-init,
  # and what it prints is published to Secret Manager, never back to hyper
  - id: create-control-plane
    definition: cluster-machines
    operation: create_control_plane
    target: gce-prod
    bound: 1
    args:
      project: hyper-example
      zone: europe-west1-b
      name: cp-1
      machine_type: zones/europe-west1-b/machineTypes/e2-standard-4
      image: projects/hyper-example/global/images/family/k8s-node-1-33
      disk_gb: "100"
      network: global/networks/hyper-example
      static_ip: 34.76.191.204
      service_account: cluster-node@hyper-example.iam.gserviceaccount.com
      cluster: hyper-example
      user_data: |
        #cloud-config
        write_files:
          - path: /etc/kubernetes/kubeadm.yaml
            permissions: "0600"
            content: |
              apiVersion: kubeadm.k8s.io/v1beta4
              kind: InitConfiguration
              localAPIEndpoint:
                bindPort: 443
              ---
              apiVersion: kubeadm.k8s.io/v1beta4
              kind: ClusterConfiguration
              kubernetesVersion: v1.33.4
              controlPlaneEndpoint: k8s.hyper-example.dev:443
          - path: /etc/kubernetes/observer.yaml
            permissions: "0600"
            content: |
              apiVersion: v1
              kind: ServiceAccount
              metadata:
                name: hyper-observer
                namespace: kube-system
              ---
              apiVersion: rbac.authorization.k8s.io/v1
              kind: ClusterRole
              metadata:
                name: hyper-observer
              rules:
                - apiGroups: [""]
                  resources: [nodes]
                  verbs: [get, list]
              ---
              apiVersion: rbac.authorization.k8s.io/v1
              kind: ClusterRoleBinding
              metadata:
                name: hyper-observer
              roleRef:
                apiGroup: rbac.authorization.k8s.io
                kind: ClusterRole
                name: hyper-observer
              subjects:
                - kind: ServiceAccount
                  name: hyper-observer
                  namespace: kube-system
        runcmd:
          - [kubeadm, init, --config, /etc/kubernetes/kubeadm.yaml]
          - [kubectl, --kubeconfig=/etc/kubernetes/admin.conf, apply, -f, /etc/kubernetes/observer.yaml]
          - [sh, -c, "kubeadm token create --print-join-command >/run/join"]
          - [sh, -c, "kubectl --kubeconfig=/etc/kubernetes/admin.conf create token hyper-observer -n kube-system --duration=24h >/run/observer"]
          - [gcloud, secrets, versions, add, kubeadm-join, --data-file=/run/join]
          - [gcloud, secrets, versions, add, cluster-observer, --data-file=/run/observer]
          - [gcloud, secrets, versions, add, cluster-ca, --data-file=/etc/kubernetes/pki/ca.crt]
          - [shred, -u, /run/join, /run/observer]

  # three workers, one Expansion; every one of them fetches the join command
  # from Secret Manager with the identity the instance already has
  - id: create-workers
    definition: cluster-machines
    operation: create_worker
    target: gce-prod
    over:
      values:
        - node-1
        - node-2
        - node-3
    bound: 3
    args:
      project: hyper-example
      zone: europe-west1-b
      name: {item: $}
      machine_type: zones/europe-west1-b/machineTypes/e2-standard-4
      image: projects/hyper-example/global/images/family/k8s-node-1-33
      disk_gb: "100"
      network: global/networks/hyper-example
      service_account: cluster-node@hyper-example.iam.gserviceaccount.com
      cluster: hyper-example
      user_data: |
        #cloud-config
        runcmd:
          - [sh, -c, "gcloud secrets versions access latest --secret=kubeadm-join >/run/join"]
          - [sh, -c, "sh /run/join"]
          - [shred, -u, /run/join]

  # the paired read: what the cluster itself says its membership is
  - id: observe-cluster-membership
    definition: cluster-membership
    operation: list_nodes
    target: cluster-api
```

No `cadence:`. Forming a cluster is not a recurrence, and the Procedure that does it is run by hand;
a repository declaring one and holding no generated workflow beside it is `projection-stale` (§10), and
there is nothing here for one to be stale against.

**The order is the meaning and there is no edge saying so** (ADR-0002). `create-workers` runs after
`create-control-plane` because it is written after it, and the join command the workers fetch exists
because the Step above published it — through Secret Manager, at a moment `hyper` does not observe and
would have no way to. Nothing in the Procedure waits for the control plane to be ready; the
`skip-if-recorded` Steps above it record that a create call was accepted, which is a different claim.

`{item: $}` in `name:` makes the `values:` list a list of **identifiers** rather than hosts: a member is
a host only where the Step wires it into an Operation's `host-input:`, and neither Manifest has one
(§3, ADR-0024). It also satisfies §4's collision check — the reference reaches the value that fills the
identity hole, so the three members cannot project one name.

`bound: 3` on the expanding `mutate` is optional and is written anyway, which buys the offline check:
`check` reads the authored list's length against it (`bound-exceeded`, §4). `bound: 1` on
`create-control-plane` buys nothing offline — a Step with no selector is invoked once — and is written
because a `mutate` without one draws an `UNBOUNDED` flag in the review, and a flag standing over a Step
that touches one thing is a reviewer's attention spent on nothing.

The three comments are source. They render verbatim in place and `hyper` never reads one (§3, §8).

**Every credential in this file is absent from it.** The join command is fetched by a machine using an
identity the machine already has; the two bearer tokens are `env:` slots on the Target declarations; and
the only thing the Procedure says about a secret is the sentence in a `runcmd` telling a machine to go
and get one.

### `hyper.yaml`

```yaml
kind: repository-declaration
version: 0.0.3-alpha
digest: sha256:e319f41282b764889447d32fa9dfcce9951e01101f6078b5bc09a035d7424eab
retention: 180d
```

Three keys, and the artefact admits only facts that govern the repository as a whole and belong to no
Procedure, Definition or Target declaration (§3): the pin, its digest, and the retention policy that
bounds Compaction. The `version:` and `digest:` pair was written by `hyper project`, which is their
only writer ([ADR-0020](../adr/0020-the-hyper-version-is-pinned-by-the-repository.md)): the digest is
the checksum published under the release tag for the artefact a runner would fetch, resolved once,
attended, and frozen into a reviewed fact.

### The file that is not an artefact

The repository on disk holds a ninth file, and it is not one of the five kinds. `AGENTS.md` is the
third thing `hyper project` writes, and it is **created or left alone** — written where the repository
holds none, and never touched again on any run (§9). It appeared here on the first `project` and is
`hyper`'s own orientation text at the version of the binary that ran, rather than anything authored:
the same text the MCP `instructions` carry, so the two channels cannot disagree about what a cold-start
agent is told.

It carries no authority, no Run reads it, and `check` does not count it — the count below is of
artefacts, and this is not one.

### Every blob id

Computed with `git hash-object` over the files exactly as they appear above. Part two's Provenance
reads its `procedure_revision` and `definition_revision` from this table.

| file | blob id |
| --- | --- |
| `providers/gce.yaml` | `2e3b8dd57dc339b996ba91f62bc30f98d1c77004` |
| `providers/kubernetes.yaml` | `152f4592fb119d516ac734062fefe63c61cfd301` |
| `targets/gce-prod.yaml` | `a4ab0d40f20c9fbc0d35045323cdb6b2e1b04419` |
| `targets/cluster-api.yaml` | `b8a9ddafe01cb48b825af58824bc566caefd4cc3` |
| `definitions/cluster-machines.yaml` | `65968929dccfefcdff7457f36f1e3d5b1bbcbf13` |
| `definitions/cluster-membership.yaml` | `2e271a684ae9f1053ee9751dcd450776f7370c6f` |
| `procedures/form-cluster.yaml` | `03b5ff935a019a9f774b4df67d39790f6d2d0c4f` |
| `hyper.yaml` | `c282e7c1270049643be1107c0720318f7b7cded8` |

A Manifest's digest is a SHA-256 over the same bytes rather than a git object id, and the two above
were read back off `hyper providers` as well as computed with `sha256sum` — they agree.

### The repository checks clean

Against a binary built from this tree and stamped with the version the pin names:

```
$ go build -ldflags "-X github.com/TheLoomLabs/hyper/internal/version.Version=0.0.3-alpha" \
    -o hyper ./cmd/hyper
$ hyper check --repo-dir cluster-repo
checked 9 artefacts: no problems found
$ echo $?
0
```

**Nine artefacts, eight artefact files.** The ninth artefact is the built-in `shell` Provider, which
ships inside the binary and has no file at all (§11); the `AGENTS.md` above is a file and not an
artefact, so it moves neither number. `hyper providers` lists the built-in beside the two authored
Manifests, both of which read `extension` — a Provider authored by someone other than `hyper` itself,
whether or not it was fetched, which is what a locally authored Manifest is.

### The negative controls

*`check` accepts this* and *`check` never looks here* are the same observation until one of them is
made false. Each row below is this repository with one edit, and the first line `check` answered with.
Every one of them was run.

| the edit | what `check` said |
| --- | --- |
| `method: POST` → `method: "{verb}"` | `hole-illegal` — *method: admits no hole of any kind* |
| `kinds: [mutate]` → `kinds: [read, mutate]` on `cluster-machines` | `definition-kinds-mixed` |
| `identity: "{name}"` → `identity: $.body.name` | `manifest-inconsistent` — *a skip-if-recorded test must resolve before the call* |
| `class: kubernetes` → `class: k8s` on `cluster-api` | `target-class-mismatch` |
| `kinds: [read]` → `kinds: [mutate]` on `cluster-api` | `envelope-exceeded`, then `kind-not-granted` |
| `bound: 3` → `bound: 2` on `create-workers` | `bound-exceeded` — *carries 3 members* |
| `node-3` → a second `node-2` | `record-identity-collision` |
| `static_ip` deleted from the input schema | `unknown-key` at the Step, `hole-illegal` at the body |
| `concurrency: 3` added to a `mutate` | `manifest-inconsistent` — *declared on an Operation that is not a read* |
| `user_data` retyped `{type: object}` | `manifest-inconsistent` — *a hole fills a scalar position only* |
| grant `*.googleapis.com`, host written literally | `host-not-granted` — the glob matched nothing |

The last row is the one worth reading twice. A wildcard in a `hosts:` grant is not a pattern that
matches a host; it is a host, and it matches nothing. That is what *no wildcard available* means as a
mechanism rather than as a rule somebody remembered to enforce.

One edit did **not** produce a fault, and it is recorded because a negative control that fails to fire
is evidence too: changing `hosts:` on `gce-prod` to a host the Manifest never names leaves `check`
clean, because `host: "{from-target}"` expands to whatever the grant holds and the intersection is
never empty. There is nothing there for `check` to catch, and nothing wrong with that — the grant *is*
the candidate set. It is the reason the wildcard row above writes the host literally.

---

## What part two adds

Part two walks the Run against these artefacts and freezes the document: every Record version with its
Provenance, every Journal entry with its Step's Disposition, the Comparison the next Run would render,
the `review` surface and the `FLAGS` its Steps draw, the join secret's path shown crossing
machine-to-machine, the `SSL_CERT_FILE` route exercised against the formed cluster with its honest
limit stated, and the paired `read`'s Observation rendered beside the effectful Step's Asset.

Then the findings — every place the corpus did not decide the question, or decided it in two places
differently — and then the freeze, at one commit, never updated.
