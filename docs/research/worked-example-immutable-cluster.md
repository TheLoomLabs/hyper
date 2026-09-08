# A worked scenario: an immutably provisioned Kubernetes cluster

**Frozen at `hyper` commit `3825d73`, worked 2026-09-04 to 2026-09-08.** This document is never
updated — it is a conformance exercise against the specification at one revision, and its value is that
it records what that revision did and did not decide. If the spec has moved on, that is expected;
re-work the scenario from scratch rather than editing this file. §11 says what the freeze covers.

One repository and three Runs, end to end — plus a second, small repository that drives the trust
route the first cannot (§9): the eight reviewed artefacts (§0–§1), every Store file the Runs write,
every rendering they produce, and the findings they left (§2–§10). Worked against `hyper`
**0.0.3-alpha** — the artefacts authored against this tree at `e599c35` and the Runs driven against it
at `3825d73`, with nothing in the specification moving between them. Everything derivable is derived —
the Manifest digests, the blob ids, the identity-set digests, the `check` counts and the `git diff`
count were computed from the actual files and read back off an actual Store, not invented, so the
numbers agree with each other and can be re-checked.

## Method, and what this is evidence of

This is not research against an external source. It is the specification read as an implementer would
read it — [`docs/spec/`](../spec/) §0–§13, [`CONTEXT.md`](../../CONTEXT.md) and the
[ADRs](../adr/) — and then *used*, once, on a scenario nobody had written down. The scenario was chosen
to make several rules meet rather than to exercise any one of them.

What §0–§1 are evidence of is one claim and no more: **the artefacts an immutably provisioned cluster
needs are authorable against the shipped product, with no new Capability, no new artefact key and no
code.** The claim is driven rather than reasoned. The eight files in §1 were written into a scratch
repository and `hyper check` was run against a binary built from this tree; every rule the prose below
says bit was made to bite, by breaking the artefact in one place and reading the code back. The
negative controls are in §1's last subsection for that reason: without them *`check` accepts this* and
*`check` never looks here* are the same observation.

What they are **not** evidence of is that the Run composes, or that the Comparison renders anything
worth reading. That is §2–§9's, and §10's findings are the deliverable of the whole exercise — here, as
in the Tailscale one, everything above the findings is a demonstration and the findings are the result.
Nothing here is normative: the specification is `docs/spec/`, the rationale is `docs/adr/`.

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

**The endpoint answers on 443, and it did not have to.** The scheme is `https` with no second one
([ADR-0082](../adr/0082-the-scheme-is-https-and-there-is-no-second-one.md)), and this exercise read
§3's `hosts:` — *a list of hosts* — as admitting no port, so the `kubeadm` configuration in the Step
below sets `bindPort: 443` and a `controlPlaneEndpoint` on the same port to meet a constraint `hyper`
was thought to impose. **It does not impose one.** §9 drives a Target granting `localhost:<port>`
through a Run of the shipped binary, and the port reaches the wire in the `Host` header; the corpus
says nothing either way, which is finding **#10**. The artefacts here are left as they were written —
binding 443 is a perfectly ordinary thing for a control plane to do — and the reasoning that produced
them is corrected rather than the files.

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
  decisive property. §9 exercises that route and states its honest limit — which the driving found to
  be narrower than the ADR's sentence.

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
absent, so this is a locally authored Provider and `origin_digest` is absent from every Provenance the
Runs below write.

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

`k8s.hyper-example.dev` carries no port because this endpoint answers on 443 (§0), and no scheme,
because there is only one. What the Target does *not* carry is the root that verifies it — a PEM in a
reviewed artefact would be a blob a reviewer cannot check, and `SSL_CERT_FILE` is where trust lives
instead (ADR-0105).

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

**The order is the meaning and there is no edge saying so** ([ADR-0002](../adr/0002-a-procedure-is-a-sequence-not-a-graph.md)). `create-workers` runs after
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

Computed with `git hash-object` over the files exactly as they appear above. The Provenance in §4 reads
its `procedure_revision` and `definition_revision` from this table.

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

## 2. How the Run was driven, and what that makes it evidence of

The Run below is a **real Run of the shipped binary**, not a paper walk. `cli.Main` was handed the
argv, the repository of §1 as a real git repository, and a process value; it ran `check` again at Run
start, resolved two credentials out of the environment, made five HTTPS requests over real handshakes,
parsed real responses, projected Records out of them and committed them to a `hyper-store` branch. Every
byte quoted in §4, §5 and §6 was read back off that branch, and every rendering in §7 is what the
command wrote to its stream.

**Four things are the fixture's, and only four.** Three of them are the reads the suite's own golden
corpus supplies for every case that reaches the world; the fourth is the one that matters to §9, and is
stated here with them rather than under it:

- **name resolution.** One in-process TLS server stands for both hosts, its certificate minted for the
  case, and the dialer `cli.Main` is handed maps `compute.googleapis.com` and `k8s.hyper-example.dev`
  to it. The handshake, the status line, the headers and the JSON parse are all real; what is supplied
  is which address a name resolves to.
- **what the world answers.** One response per request, in order, authored in the shapes the two APIs
  document: a `compute#operation` job object for each `instances.insert`, and a `NodeList` for each
  read of `/api/v1/nodes`. This is the class of error §4 states it has no oracle for — if GCE's insert
  answers something other than a zone operation, the Manifest is wrong and nothing here would notice.
- **the clock, the Run ids, and who ran it.** A Run reads the clock many times — fifteen for each of
  the first two Runs here and fourteen for the third — and the fixture's advances 100ms per read, so
  the intervals below are the fixture's and the *sequence*, which file carries which instant, is
  `hyper`'s. The three Run ids are stated by the case, which is what makes
  every Store path a constant. The actor and host are the corpus's own `igor@thinkpad`.
- **the trust anchor.** The harness's dialer carries the pool the server's certificate was minted
  against, which is how a case reaches a host nothing public signed for. **So these Runs did not
  exercise `SSL_CERT_FILE`**, and §9 does not pretend they did: it drives that route separately, with
  the shipped binary against a stand-in for the cluster's API, and says exactly what it drove.

What that makes this: evidence about **what `hyper` does with an answer**, and no evidence at all about
what Google Compute Engine or a `kubeadm` cluster would have answered. The distinction is the same one
part one drew about `check`, one layer out.

The scratch repository, the case, the driver and the whole transcript are session material rather than
part of this tree. Rebuilding them is small: a case directory carrying the §1 repository, a `serve/`
entry per host, an `env`, a `now` and a `mint`, and a driver against `internal/cli`'s own helpers —
`corpusCase`, `invocation`, `process` and `cli.Main` — with the process's `Now` replaced per Run, so
that three Runs stand a day apart against one fixture.

### What the Store holds before the Run

Nothing. One `hyper-store` branch holding one commit and one file — `STORE.md`, fourteen bytes — which
is the state `hyper store init` leaves and the state every first Run of a repository begins in. There
is no baseline Run, so there is nothing to compare until the first Run has closed, and a `review` of
the Procedure opens on `no baseline — form-cluster has not run` until it has. That is a different named
state from `no baseline — no Store`, which is what the same command answers in a repository with no
Store branch at all — the two absences are told apart rather than collapsed (§8).

---

## 3. What the first Run does

`hyper run form-cluster`, at `2026-09-04T08:31:06.512Z`. Run id
`01a06b8a-feac-7d31-b4a2-6f0c19e85d47` — a UUIDv7 whose 48-bit prefix is that instant — trigger
manual, executor local.

**Step 1 `create-control-plane`.** No selector, so a set of one and one `skip-if-recorded` test. The
Store holds no series under `(gce-prod, cluster-machines, cp-1)`, so the test concludes *not recorded*
and the call goes out: `POST /compute/v1/projects/hyper-example/zones/europe-west1-b/instances`, with
the whole cloud-init in the body's `metadata.items[0].value`. The answer is a zone operation, and the
four projected fields come off it. Disposition **ran**, one identity.

**Step 2 `create-workers`.** The Expansion is the authored list, top-first, and an effectful Expansion
is strictly serial (ADR-0045), so three calls go out in the order the file writes them. Each member
runs its own skip test at its own turn (ADR-0056); none is recorded; three calls, three Assets.
Disposition **ran**, three identities.

**Step 3 `observe-cluster-membership`.** A `read` against the cluster's own API. It runs seconds after
the four create calls were accepted, and what the API server knows at that moment is **one node** — the
control plane, which formed the cluster from its own cloud-init; the three workers are still booting.
The Step projects a series of one. Disposition **ran**, one identity.

Outcome `completed`, exit `0`.

That third Step is the first thing this exercise found that no amount of reading would have produced:
the paired `read` is paired with an effect it cannot yet see. It is finding **#8**.

### The page

```
STEP  ID                          KIND    DISPOSITION  RECORDS
1     create-control-plane        mutate  ran          1
2     create-workers              mutate  ran          3
3     observe-cluster-membership  read    ran          1

completed · exit 0 · run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47
```

The `RECORDS` column is the size of each Step's identity set — what the Step concluded about — and
here it is also what each Step wrote, no Record having come back unchanged on a first Run. Nothing
renders before the Run: no summary, no confirmation, on any Kind ([ADR-0015](../adr/0015-the-cli-never-prompts.md)).

The progress lines go to **stderr** and the page to stdout, which is what lets the page be piped
without losing the narration:

```
run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47
step 1/3 create-control-plane
step 2/3 create-workers
step 3/3 observe-cluster-membership
```

---

## 4. The Store files the first Run writes

Ten paths, every one of them new — the eleventh file on the branch is the `STORE.md` that was
already there. Nothing in the Store is ever rewritten
([ADR-0011](../adr/0011-the-store-is-append-only.md)), and every path carries the id of the Run that
wrote it ([ADR-0076](../adr/0076-every-store-path-carries-the-id-of-the-run-that-wrote-it.md)). No
segment needs percent-encoding — every name is `[a-z0-9-]` — and none approaches the 200-byte
truncation (§12).

```
journal/2026/09/04/01a06b8a-feac-7d31-b4a2-6f0c19e85d47/outcome.json
journal/2026/09/04/01a06b8a-feac-7d31-b4a2-6f0c19e85d47/run.json
journal/2026/09/04/01a06b8a-feac-7d31-b4a2-6f0c19e85d47/steps/0001.json
journal/2026/09/04/01a06b8a-feac-7d31-b4a2-6f0c19e85d47/steps/0002.json
journal/2026/09/04/01a06b8a-feac-7d31-b4a2-6f0c19e85d47/steps/0003.json
records/cluster-api/cluster-membership/cp-1/01a06b8a-feac-7d31-b4a2-6f0c19e85d47-0003.json
records/gce-prod/cluster-machines/cp-1/01a06b8a-feac-7d31-b4a2-6f0c19e85d47-0001.json
records/gce-prod/cluster-machines/node-1/01a06b8a-feac-7d31-b4a2-6f0c19e85d47-0002.json
records/gce-prod/cluster-machines/node-2/01a06b8a-feac-7d31-b4a2-6f0c19e85d47-0002.json
records/gce-prod/cluster-machines/node-3/01a06b8a-feac-7d31-b4a2-6f0c19e85d47-0002.json
```

**The `<nnnn>` suffix is the Step position**, not a counter of versions: `cp-1` under
`cluster-machines` is `-0001` because Step 1 wrote it, the three workers are `-0002` because Step 2
did, and the Observation of `cp-1` is `-0003`. Two Steps of one Run writing one identity would write
two paths rather than one path twice, which is what that segment disambiguates (§12).

**`cp-1` appears twice, under two Targets and two Definitions, and the two are unrelated.** One is
*a create call was accepted for a machine with this name*; the other is *the API server knows a node
with this name*. `hyper` asserts no relation between them and the path is what says so — the
Definition is a segment of a Record's identity (ADR-0032).

### The commits

One commit per Record version, one per Step file, and a Begin and an End around them — ten for this
Run, each with a message a human reads in `git log`. The eleventh line below is the branch's own seed
commit and is not this Run's.

```
5b64e7c89845344763e15a5c2e44f5eda7dde622 2026-09-04T08:31:06Z End run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47
d6f2fc0b245ded725749721ccba0c31ee658fc8d 2026-09-04T08:31:06Z Step 3 observe-cluster-membership of run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47: ran
a8a90d5da7862cf5ccee0cd88d76c81c2be04907 2026-09-04T08:31:06Z Record cluster-api/cluster-membership/cp-1 at run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47 step 3
28550d5b11f645f7ee79fd1c77f231e8b28ac365 2026-09-04T08:31:06Z Step 2 create-workers of run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47: ran
cd13f172d7ee1e0b0eb57384b4e25c4370eb9035 2026-09-04T08:31:06Z Record gce-prod/cluster-machines/node-3 at run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47 step 2
b2955f33a1a5199975c4269b6691cce1cbdacff3 2026-09-04T08:31:06Z Record gce-prod/cluster-machines/node-2 at run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47 step 2
6b80cee754d8c2ecfa5bb4a369cc862912d458c6 2026-09-04T08:31:06Z Record gce-prod/cluster-machines/node-1 at run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47 step 2
3433ca895328c390fde0b78d5611e924731cfce3 2026-09-04T08:31:06Z Step 1 create-control-plane of run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47: ran
30601b10efbec4da710769f248025540c523927c 2026-09-04T08:31:06Z Record gce-prod/cluster-machines/cp-1 at run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47 step 1
5a2c369aa0fdec08627a950f1f4b16565ce16a49 2026-09-04T08:31:06Z Begin run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47
75549218640224b52a48ba140c4425ff85beb7b6 2026-09-04T08:31:06Z the fixture's hyper-store
```

Every commit carries the Run's **start** instant, truncated to whole seconds, while the Journal's own
`started_at`, `ended_at` and `written_at` advance through the Run. Both are the clock this process was
handed; a git date is whole seconds and the sub-second half is the entry's (finding **#12**).

### `run.json`

```json
{
  "dry_run": false,
  "procedure": "form-cluster",
  "provenance": {
    "hyper_version": "0.0.3-alpha",
    "procedure_revision": "03b5ff935a019a9f774b4df67d39790f6d2d0c4f",
    "repo_revision": "43918fcac50ab84d5638c99caaa3a275c0eb26d2"
  },
  "run_id": "01a06b8a-feac-7d31-b4a2-6f0c19e85d47",
  "schema_version": 1,
  "started_at": "2026-09-04T08:31:06.512Z",
  "trigger": {
    "actor": "igor",
    "cause": "manual",
    "executor": "local",
    "host": "thinkpad"
  }
}
```

`dry_run: false` is written out — the one marker in the Store that does not follow the absence rule
(§7). Run-level Provenance carries only the members that have exactly one value at Run level, so
there is no `definition_revision` and no `manifest_digest` here
([ADR-0043](../adr/0043-a-provenance-member-is-written-where-it-has-exactly-one-value.md)); no
`repo_dirty`, the tree being clean; and no `origin_digest` anywhere in this Run, both Manifests
being locally authored.

### `steps/0001.json` — `create-control-plane`

```json
{
  "definition": "cluster-machines",
  "disposition": "ran",
  "ended_at": "2026-09-04T08:31:06.912Z",
  "id": "create-control-plane",
  "identities": {
    "digest": "sha256:f7b3085a922910ffbd9a68980b0e463b2edf274fb825f277ffb3c3749fcd84c1",
    "members": [
      "cp-1"
    ]
  },
  "kind": "mutate",
  "operation": "create_control_plane",
  "provenance": {
    "definition_revision": "65968929dccfefcdff7457f36f1e3d5b1bbcbf13",
    "manifest_digest": "sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84"
  },
  "provider": "gce",
  "schema_version": 1,
  "started_at": "2026-09-04T08:31:06.712Z",
  "step": 1,
  "target": "gce-prod"
}
```

`members` is written because there is no earlier Run of this Procedure for the digest to have been
carried by, so the digest necessarily moved
([ADR-0055](../adr/0055-a-steps-identity-digest-is-compared-against-the-last-run-that-carried-one.md)).

The digest is re-checkable: it is `sha256:` over the canonical JSON encoding of the sorted array with
its trailing LF, and this exercise settles what those bytes are by driving rather than by reading —

```
$ printf '[\n  "cp-1"\n]\n' | sha256sum
f7b3085a922910ffbd9a68980b0e463b2edf274fb825f277ffb3c3749fcd84c1  -
```

— which is the reading the Tailscale exercise guessed at in its finding **#4**, now confirmed against
four digests rather than assumed against one (finding **#5**).

There is no `bound` in this file and the Step declares `bound: 1`, which is the first sighting of
finding **#1** — the largest thing this exercise found, and stated in full there.

### `steps/0002.json` — `create-workers`

```json
{
  "definition": "cluster-machines",
  "disposition": "ran",
  "ended_at": "2026-09-04T08:31:07.412Z",
  "id": "create-workers",
  "identities": {
    "digest": "sha256:8a41bb660b716da3ef34d93a086b6fa0cb8ad810506db2791a31ddffc00f9061",
    "members": [
      "node-1",
      "node-2",
      "node-3"
    ]
  },
  "kind": "mutate",
  "operation": "create_worker",
  "provenance": {
    "definition_revision": "65968929dccfefcdff7457f36f1e3d5b1bbcbf13",
    "manifest_digest": "sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84"
  },
  "provider": "gce",
  "schema_version": 1,
  "selector": {
    "declared": {
      "values": [
        "node-1",
        "node-2",
        "node-3"
      ]
    },
    "expanded_to": [
      "node-1",
      "node-2",
      "node-3"
    ]
  },
  "started_at": "2026-09-04T08:31:07.012Z",
  "step": 2,
  "target": "gce-prod"
}
```

`declared` and `expanded_to` are identical and in the authored order, which says no member was dropped;
`members` is the same three sorted by Unicode code point, which is the set the digest is taken over.
The Step declares `bound: 3` and the file records no Bound — again finding **#1**.

No Pattern account: `retry: {attempts: 3}` is declared on the Operation and every call was answered
first time, so a Pattern that did no more than the trivial single call writes nothing (§7).

### `steps/0003.json` — `observe-cluster-membership`

```json
{
  "definition": "cluster-membership",
  "disposition": "ran",
  "ended_at": "2026-09-04T08:31:07.712Z",
  "id": "observe-cluster-membership",
  "identities": {
    "digest": "sha256:f7b3085a922910ffbd9a68980b0e463b2edf274fb825f277ffb3c3749fcd84c1",
    "members": [
      "cp-1"
    ]
  },
  "kind": "read",
  "operation": "list_nodes",
  "provenance": {
    "definition_revision": "2e271a684ae9f1053ee9751dcd450776f7370c6f",
    "manifest_digest": "sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6"
  },
  "provider": "kubernetes",
  "schema_version": 1,
  "started_at": "2026-09-04T08:31:07.512Z",
  "step": 3,
  "target": "cluster-api"
}
```

**Its digest is Step 1's digest.** Both Steps concluded about a set whose one member is named `cp-1`,
and the digest is taken over the names alone — not over the Target, the Definition or the Kind. Two
Steps of one Run can therefore carry one digest while describing two entirely different things, which
is safe because ADR-0055's *unchanged since* resolution walks the history of **that Step**, and worth
knowing before someone treats a digest as an identifier of a set of Records (finding **#13**).

No `selector` key at all: a Step carrying no `over:` resolved none and holds none. No `answered`
either — and not because the call was `2xx`: that member is written on **no `read` at all**, a `read`'s
status being the answer rather than something that decided the Step, and belonging in the Record
wherever the Manifest projected it (§7).

### `outcome.json`

```json
{
  "ended_at": "2026-09-04T08:31:07.812Z",
  "outcome": "completed",
  "schema_version": 1
}
```

No exit code, no duration, no `refusal`. The exit code is a mapping the CLI applies and the Store does
not restate a rendering (§7); the duration is the difference between two instants a reader already has.

### The Assets

```json
{
  "definition": "cluster-machines",
  "fields": {
    "operation": "operation-1788510666512-63d1a4f0b9c2e-4f1a2b3c-9d8e7f60",
    "started": "2026-09-04T01:31:06.514-07:00",
    "status": "RUNNING",
    "target_link": "https://www.googleapis.com/compute/v1/projects/hyper-example/zones/europe-west1-b/instances/cp-1"
  },
  "name": "cp-1",
  "operation": "create_control_plane",
  "provenance": {
    "definition_revision": "65968929dccfefcdff7457f36f1e3d5b1bbcbf13",
    "hyper_version": "0.0.3-alpha",
    "manifest_digest": "sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84",
    "procedure_revision": "03b5ff935a019a9f774b4df67d39790f6d2d0c4f",
    "repo_revision": "43918fcac50ab84d5638c99caaa3a275c0eb26d2"
  },
  "record_type": "asset",
  "run_id": "01a06b8a-feac-7d31-b4a2-6f0c19e85d47",
  "schema_version": 1,
  "step": 1,
  "target": "gce-prod",
  "written_at": "2026-09-04T08:31:06.812Z"
}
```

A Record version carries the **whole** of Provenance, unlike the Step file beside it, because it sits
under a Record path with no entry next to it (ADR-0043).

**Every field here is about the call and none is about the machine.** `status: RUNNING` is the *job's*
status — an insert that has been accepted and is being worked — and `target_link` is the URL the
instance will have. Nothing in this file says a machine booted, that cloud-init ran, or that `kubeadm`
formed anything, and part one predicted exactly that from the Manifest. This is what an Asset written
against a cloud API's asynchronous create is accountable for, and it is the same shape as the
accountability limit an Opaque Operation has: the effect reached the call, and the world past it is
outside what the Record claims.

The three workers write the same shape under `-0002`, in Expansion order, at
`08:31:07.112Z`, `08:31:07.212Z` and `08:31:07.312Z` — ascending, which is what a serial effectful
Expansion guarantees.

### The Observation

```json
{
  "definition": "cluster-membership",
  "fields": {
    "created": "2026-09-04T08:31:41Z",
    "kubelet_version": "v1.33.4",
    "name": "cp-1",
    "os_image": "Ubuntu 24.04.3 LTS"
  },
  "name": "cp-1",
  "operation": "list_nodes",
  "provenance": {
    "definition_revision": "2e271a684ae9f1053ee9751dcd450776f7370c6f",
    "hyper_version": "0.0.3-alpha",
    "manifest_digest": "sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6",
    "procedure_revision": "03b5ff935a019a9f774b4df67d39790f6d2d0c4f",
    "repo_revision": "43918fcac50ab84d5638c99caaa3a275c0eb26d2"
  },
  "record_type": "observation",
  "run_id": "01a06b8a-feac-7d31-b4a2-6f0c19e85d47",
  "schema_version": 1,
  "step": 3,
  "target": "cluster-api",
  "written_at": "2026-09-04T08:31:07.612Z"
}
```

`record_type` is `observation` rather than `asset`, which is the type the *Definition's* claimed Kinds
decide and not the Operation's (§3, ADR-0032).

**`unschedulable` is absent, and that is the ordinary state.** The Manifest projects it; the path
`$.spec.unschedulable` resolves to nothing on a schedulable node; the field is simply not in `fields`.
A projection whose field path does not resolve writes no member and fails no Step — part one said so
from the specification and the Run confirms it.

**`name` appears twice**, once as the Record's `name` and once as a projected field, because the
Manifest projects `$.metadata.name` into `fields` as well as into `identity:`. Nothing forbids it and
the Comparison renders it in the `FIELDS` column like any other value; it is worth seeing once to know
that the identity is *not* implicitly among the fields.

---

## 5. The second Run: a fourth worker

A day later the fleet grows. An agent adds `node-4` to the Step's `values:` list and raises the Bound
from 3 to 4 — two lines in one file — and a human reviews the diff and commits it. The Procedure's blob
moves from `03b5ff935a019a9f774b4df67d39790f6d2d0c4f` to
`ed503f1599a9f02ecd4e64677fc4c3f9ff17b52c`, and nothing else in the repository moves at all.

`hyper run form-cluster`, at `2026-09-05T09:47:31.984Z`. Run id
`01a070f7-52ac-7a19-9e63-2c81f4b0da95`.

```
STEP  ID                          KIND    DISPOSITION                  RECORDS
1     create-control-plane        mutate  skipped-as-already-recorded  1
2     create-workers              mutate  ran                          4
3     observe-cluster-membership  read    ran                          4

completed · exit 0 · run 01a070f7-52ac-7a19-9e63-2c81f4b0da95
```

**Step 1 skipped and still reports a Record.** The head version of `(gce-prod, cluster-machines, cp-1)`
stands, so the skip test concluded about it and made no call — which is the one Disposition that is
Repeatability evidence, and it carries a set of one (§7, ADR-0056).

**Step 2 ran, and three of its four members skipped.** `skip-if-recorded` decides per Record and not
per Step, so the Expansion tests each member at its own turn: `node-1`, `node-2` and `node-3` have
heads that stand and are skipped; `node-4` has no series at all and is called for. One call went out,
so the Step is *ran* and not *skipped as already recorded* — that value is for a Step where **every**
member skipped. The column reads `4` and not `1 of 4`: the skip test reached a conclusion about every
member, so nothing is unaccounted for (§8).

**Step 3 ran and found four nodes.** The three workers have joined since the first Run, so the paired
`read` now records what the cluster says its membership is — and `node-4`, whose create call the Step
above it made a moment earlier, is not among them.

### The Step files

```json
{
  "definition": "cluster-machines",
  "disposition": "skipped-as-already-recorded",
  "ended_at": "2026-09-05T09:47:32.284Z",
  "id": "create-control-plane",
  "identities": {
    "digest": "sha256:f7b3085a922910ffbd9a68980b0e463b2edf274fb825f277ffb3c3749fcd84c1"
  },
  "kind": "mutate",
  "operation": "create_control_plane",
  "provenance": {
    "definition_revision": "65968929dccfefcdff7457f36f1e3d5b1bbcbf13",
    "manifest_digest": "sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84"
  },
  "provider": "gce",
  "schema_version": 1,
  "started_at": "2026-09-05T09:47:32.184Z",
  "step": 1,
  "target": "gce-prod"
}
```

A skipped Step carries a digest and **no members**: the set is the one this Step carried in the last
Run of the Procedure that carried one, and it did not move (ADR-0055). Resolving it means walking back
to that Run, which `show` does by naming it rather than rendering the members bare.

```json
{
  "definition": "cluster-machines",
  "disposition": "ran",
  "ended_at": "2026-09-05T09:47:32.584Z",
  "id": "create-workers",
  "identities": {
    "digest": "sha256:b23bc5b47f21666498ed8c277a12af0ae7655c0fe15cdd2d16624e88e14054ef",
    "members": [
      "node-1",
      "node-2",
      "node-3",
      "node-4"
    ]
  },
  "kind": "mutate",
  "operation": "create_worker",
  "provenance": {
    "definition_revision": "65968929dccfefcdff7457f36f1e3d5b1bbcbf13",
    "manifest_digest": "sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84"
  },
  "provider": "gce",
  "schema_version": 1,
  "selector": {
    "declared": {
      "values": [
        "node-1",
        "node-2",
        "node-3",
        "node-4"
      ]
    },
    "expanded_to": [
      "node-1",
      "node-2",
      "node-3",
      "node-4"
    ]
  },
  "started_at": "2026-09-05T09:47:32.384Z",
  "step": 2,
  "target": "gce-prod"
}
```

The digest moved because the **artefact** gained a member, not because the world did:
`sha256:8a41bb…` for three, `sha256:b23bc5…` for four. `declared` and `expanded_to` hold all four
including the three that skipped, which is the whole reason the set is *what the Step concluded
about* and not *what it wrote*
([ADR-0030](../adr/0030-a-steps-identity-set-holds-what-it-concluded-not-what-it-wrote.md)).

```json
{
  "definition": "cluster-membership",
  "disposition": "ran",
  "ended_at": "2026-09-05T09:47:33.184Z",
  "id": "observe-cluster-membership",
  "identities": {
    "digest": "sha256:f4a36f41e67cd9e20cd2ad5a6b9658bb31edb2f771357e07f052b1d6309eebce",
    "members": [
      "cp-1",
      "node-1",
      "node-2",
      "node-3"
    ]
  },
  "kind": "read",
  "operation": "list_nodes",
  "provenance": {
    "definition_revision": "2e271a684ae9f1053ee9751dcd450776f7370c6f",
    "manifest_digest": "sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6"
  },
  "provider": "kubernetes",
  "schema_version": 1,
  "started_at": "2026-09-05T09:47:32.684Z",
  "step": 3,
  "target": "cluster-api"
}
```

Three of the four nodes here are new to the record and `cp-1` is not, and the Step file says nothing
about which is which — that is the Comparison's to render.

### What the second Run wrote

Four Record versions and no more: `node-4`'s Asset, and Observations for `node-1`, `node-2` and
`node-3`. **`cp-1`'s Observation was not rewritten**, because the API server answered exactly what it
answered a day earlier and identical bytes mint no version. The Comparison is what tells *nothing
changed* from *nobody looked*, and the identity set is what makes it able to.

The three Observations are the shape §4's is, under this Run's Provenance and at `-0003`, differing
only in their names, their `created` and their `written_at` — `09:47:32.884Z`, `09:47:32.984Z` and
`09:47:33.084Z`, ascending in the order the collection came back. The Asset:

```json
{
  "definition": "cluster-machines",
  "fields": {
    "operation": "operation-1788601652004-63d3f1a08c7d5-5e6f7a8b-9c0d1e20",
    "started": "2026-09-05T02:47:32.006-07:00",
    "status": "RUNNING",
    "target_link": "https://www.googleapis.com/compute/v1/projects/hyper-example/zones/europe-west1-b/instances/node-4"
  },
  "name": "node-4",
  "operation": "create_worker",
  "provenance": {
    "definition_revision": "65968929dccfefcdff7457f36f1e3d5b1bbcbf13",
    "hyper_version": "0.0.3-alpha",
    "manifest_digest": "sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84",
    "procedure_revision": "ed503f1599a9f02ecd4e64677fc4c3f9ff17b52c",
    "repo_revision": "e81f88383d44df5aee872e299cdf1a3ab84dac38"
  },
  "record_type": "asset",
  "run_id": "01a070f7-52ac-7a19-9e63-2c81f4b0da95",
  "schema_version": 1,
  "step": 2,
  "target": "gce-prod",
  "written_at": "2026-09-05T09:47:32.484Z"
}
```

---

## 6. The third Run: nothing left to do

The steady state of an immutably provisioned cluster, and the Run an operator will see most often. No
artefact moved between the second Run and this one.

`hyper run form-cluster --json`, at `2026-09-05T14:12:44.401Z`. Run id
`01a0721c-8f30-7b52-8d17-53ae60b9c4f2`.

```
{"type":"step","step":1,"id":"create-control-plane","kind":"mutate","disposition":"skipped-as-already-recorded","records":1}
{"type":"step","step":2,"id":"create-workers","kind":"mutate","disposition":"skipped-as-already-recorded","records":4}
{"type":"step","step":3,"id":"observe-cluster-membership","kind":"read","disposition":"ran","records":4}
{"type":"provenance","hyper_version":"0.0.3-alpha","procedure_revision":"ed503f1599a9f02ecd4e64677fc4c3f9ff17b52c","repo_revision":"e81f88383d44df5aee872e299cdf1a3ab84dac38"}
{"type":"provenance","step":1,"definition_revision":"65968929dccfefcdff7457f36f1e3d5b1bbcbf13","manifest_digest":"sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84"}
{"type":"provenance","step":2,"definition_revision":"65968929dccfefcdff7457f36f1e3d5b1bbcbf13","manifest_digest":"sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84"}
{"type":"provenance","step":3,"definition_revision":"2e271a684ae9f1053ee9751dcd450776f7370c6f","manifest_digest":"sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6"}
{"type":"outcome","outcome":"completed","code":0,"dry_run":false,"run_id":"01a0721c-8f30-7b52-8d17-53ae60b9c4f2"}
```

Both effectful Steps skipped every member; the `read` ran and found the same four nodes it found
before. **The Run wrote five Journal files and not one Record version** — a `read` that observes
nothing new writes nothing at all, and a reader who expects one file per conclusion would read this
entry as truncated. It is not: the conclusions are the identity sets on the three Step files.
Every revision and digest is written **whole** on the wire
([ADR-0047](../adr/0047-an-id-a-human-retypes-renders-whole.md)); one `provenance` row per Step file
and one Run-wide row, which is the Store's own split arriving on the stream; and the `outcome` row
carries no `error_code`, a completed Run having no check to name. 
The Comparison this Run and its baseline render is the one an operator reads to confirm that a fleet is
where it was left:

```
  form-cluster
  BASELINE  01a070f7-52ac…  igor@thinkpad  Sat 5 Sep 09:47  completed  1s  procedure rev ed503f1
  SUBJECT   01a0721c-8f30…  igor@thinkpad  Sat 5 Sep 14:12  completed  1s  procedure rev ed503f1

  YOU DID THIS   0 assets

  THE WORLD MOVED   0 observations

  THE CODE MOVED   0 facts
  0 other lines changed · git diff e81f883 e81f883

  TOTALS  0 changes · 0 asset · 0 observation · 0 tombstone · the code did not move
```

All three tables render their header and their count with no row beneath and **no column-header row**,
which answers the Tailscale exercise's finding **#14** by driving it. `THE CODE MOVED` renders its
catch-all against a revision and itself, and `TOTALS` ends *the code did not move*.

---

## 7. The renderings

Every rendering below was taken **after the second Run and before the third**, which is the state an
operator is in when they come back to a repository: two Runs in the record and a Procedure that has
just grown a machine.

### `hyper changes form-cluster`

The Comparison the second Run renders against the first: what `hyper` did, what the world did, and what
the code did, in that order and on one screen.

```
  form-cluster
  BASELINE  01a06b8a-feac…  igor@thinkpad  Fri 4 Sep 08:31  completed  1s  procedure rev 03b5ff9
  SUBJECT   01a070f7-52ac…  igor@thinkpad  Sat 5 Sep 09:47  completed  1s  procedure rev ed503f1

  YOU DID THIS   1 asset
  CHANGE   TARGET    DEFINITION        RECORD  ORDINAL  FIELDS
  created  gce-prod  cluster-machines  node-4  – → 1    operation: operation-1788601652004-63d3f1a08c7d5-5e6f7a8b-9c0d1e20 · started: 2026-09-05T02:47:32.006-07:00 · status: RUNNING · target_link: https://www.googleapis.com/compute/v1/projects/hyper-example/zones/europe-west1-b/instances/node-4

  THE WORLD MOVED   3 observations
  CHANGE    TARGET       DEFINITION          RECORD  ORDINAL  FIELDS
  appeared  cluster-api  cluster-membership  node-1  – → 1    created: 2026-09-04T08:33:02Z · kubelet_version: v1.33.4 · name: node-1 · os_image: Ubuntu 24.04.3 LTS
  appeared  cluster-api  cluster-membership  node-2  – → 1    created: 2026-09-04T08:33:14Z · kubelet_version: v1.33.4 · name: node-2 · os_image: Ubuntu 24.04.3 LTS
  appeared  cluster-api  cluster-membership  node-3  – → 1    created: 2026-09-04T08:33:29Z · kubelet_version: v1.33.4 · name: node-3 · os_image: Ubuntu 24.04.3 LTS

  THE CODE MOVED   4 facts
  SUBJECT                 FACT                         FROM                      TO
  procedure form-cluster  procedure revision           03b5ff9                   ed503f1
  procedure form-cluster  step create-workers · bound  3                         4
  procedure form-cluster  step create-workers · over   values                    values
                                                       node-1 · node-2 · node-3  node-1 · node-2 · node-3 · node-4
  —                       repository revision          43918fc                   e81f883
  0 other lines changed · git diff 43918fc e81f883

  TOTALS  4 changes · 1 asset · 3 observation · 0 tombstone · the code moved
```

Four things here are worth reading twice.

**`node-1`, `node-2` and `node-3` get no asset row and three observation rows.** Their Assets were in
the subject Run's identity set and did not move — they were skipped — so `YOU DID THIS` says nothing
about them, which is how *nothing changed* is told from *nobody looked*. The rows they do have are
Observations, and they say the thing the exercise was built to show: **`hyper` did not do this**. The
machines joined the cluster by themselves, out of the cloud-init the first Run put in a request body,
and the only reason there is anything to render at all is the paired `read`.

**`cp-1` gets no row at all, on either table.** Its Asset was skipped and its Observation came back
byte-identical, so it is in two identity sets and out of every table. That is the whole of the
*unchanged* state, rendered as an absence.

**`node-4` is an Asset with no Observation.** It was created by the subject Run and the cluster does not
know it yet. A reader of this screen is being shown the exact state the immutable model produces: a
machine `hyper` is accountable for having asked for, and no evidence yet that it did anything. The
evidence arrives on the next Run, which is finding **#8**.

**`THE CODE MOVED` renders a non-scalar fact on two lines.** The `over` row puts the class — `values` —
in `FROM` and `TO`, and the members beneath them, aligned under the columns. The specification states
the `cadence` row's stacked form in full and states nothing about this one; the Tailscale exercise
raised that as its finding **#5** and invented a one-line rendering. The binary has an answer, and this
is it (finding **#4**).

The catch-all reads `0 other lines changed`: the edit was two lines, one classified by the Bounds class
and one by the selector class, so nothing is left for `git diff` to count.

### `hyper changes form-cluster --json`

```
{"type":"window","procedure":"form-cluster","baseline":{"run":"01a06b8a-feac-7d31-b4a2-6f0c19e85d47","trigger":"igor@thinkpad","started":"2026-09-04T08:31:06.512Z","dry_run":false,"outcome":"completed","ended":"2026-09-04T08:31:07.812Z","procedure_revision":"03b5ff935a019a9f774b4df67d39790f6d2d0c4f"},"subject":{"run":"01a070f7-52ac-7a19-9e63-2c81f4b0da95","trigger":"igor@thinkpad","started":"2026-09-05T09:47:31.984Z","dry_run":false,"outcome":"completed","ended":"2026-09-05T09:47:33.284Z","procedure_revision":"ed503f1599a9f02ecd4e64677fc4c3f9ff17b52c"}}
{"type":"asset","change":"created","target":"gce-prod","definition":"cluster-machines","name":"node-4","to_ordinal":1,"fields":{"operation":"operation-1788601652004-63d3f1a08c7d5-5e6f7a8b-9c0d1e20","started":"2026-09-05T02:47:32.006-07:00","status":"RUNNING","target_link":"https://www.googleapis.com/compute/v1/projects/hyper-example/zones/europe-west1-b/instances/node-4"}}
{"type":"observation","change":"appeared","target":"cluster-api","definition":"cluster-membership","name":"node-1","to_ordinal":1,"fields":{"created":"2026-09-04T08:33:02Z","kubelet_version":"v1.33.4","name":"node-1","os_image":"Ubuntu 24.04.3 LTS"}}
{"type":"observation","change":"appeared","target":"cluster-api","definition":"cluster-membership","name":"node-2","to_ordinal":1,"fields":{"created":"2026-09-04T08:33:14Z","kubelet_version":"v1.33.4","name":"node-2","os_image":"Ubuntu 24.04.3 LTS"}}
{"type":"observation","change":"appeared","target":"cluster-api","definition":"cluster-membership","name":"node-3","to_ordinal":1,"fields":{"created":"2026-09-04T08:33:29Z","kubelet_version":"v1.33.4","name":"node-3","os_image":"Ubuntu 24.04.3 LTS"}}
{"type":"code","subject_kind":"procedure","subject":"form-cluster","fact":"procedure revision","from":"03b5ff935a019a9f774b4df67d39790f6d2d0c4f","to":"ed503f1599a9f02ecd4e64677fc4c3f9ff17b52c"}
{"type":"code","subject_kind":"procedure","subject":"form-cluster","fact":"step create-workers · bound","from":3,"to":4}
{"type":"code","subject_kind":"procedure","subject":"form-cluster","fact":"step create-workers · over","from":{"values":["node-1","node-2","node-3"]},"to":{"values":["node-1","node-2","node-3","node-4"]}}
{"type":"code","fact":"repository revision","from":"43918fcac50ab84d5638c99caaa3a275c0eb26d2","to":"e81f88383d44df5aee872e299cdf1a3ab84dac38"}
{"type":"code","fact":"other lines changed","count":0,"command":"git diff 43918fc e81f883"}
{"type":"result","truncated":false}
```

`from_ordinal` is absent on the `created` and `appeared` rows exactly where the column renders `–`. The
`over` row carries the two selectors whole as objects, which is the structure the page renders stacked.
`changes` is not a Run, so the stream terminates with `result` and not `outcome`.

### `hyper show <run> --expansion`

**§8 prints no example of this surface**, which the Tailscale exercise recorded as its finding **#10**
and answered by inventing a layout. This is the layout the binary writes.

```
ENTRY                01a06b8a-feac-7d31-b4a2-6f0c19e85d47
PROCEDURE            form-cluster
CAUSE                manual
EXECUTOR             local
ACTOR                igor
HOST                 thinkpad
STARTED              2026-09-04T08:31:06.512Z
OUTCOME              completed
ENDED                2026-09-04T08:31:07.812Z
HYPER                0.0.3-alpha
PROCEDURE REVISION   03b5ff935a019a9f774b4df67d39790f6d2d0c4f
REPOSITORY REVISION  43918fcac50ab84d5638c99caaa3a275c0eb26d2

STEP 1  create-control-plane · mutate · ran
DEFINITION           cluster-machines
OPERATION            create_control_plane
PROVIDER             gce
TARGET               gce-prod
STARTED              2026-09-04T08:31:06.712Z
ENDED                2026-09-04T08:31:06.912Z
RECORDS              cp-1
DEFINITION REVISION  65968929dccfefcdff7457f36f1e3d5b1bbcbf13
MANIFEST DIGEST      sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84

STEP 2  create-workers · mutate · ran
DEFINITION           cluster-machines
OPERATION            create_worker
PROVIDER             gce
TARGET               gce-prod
STARTED              2026-09-04T08:31:07.012Z
ENDED                2026-09-04T08:31:07.412Z
RECORDS              node-1 · node-2 · node-3
SELECTOR             values · node-1 · node-2 · node-3
EXPANDED TO          node-1 · node-2 · node-3
DEFINITION REVISION  65968929dccfefcdff7457f36f1e3d5b1bbcbf13
MANIFEST DIGEST      sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84

STEP 3  observe-cluster-membership · read · ran
DEFINITION           cluster-membership
OPERATION            list_nodes
PROVIDER             kubernetes
TARGET               cluster-api
STARTED              2026-09-04T08:31:07.512Z
ENDED                2026-09-04T08:31:07.712Z
RECORDS              cp-1
DEFINITION REVISION  2e271a684ae9f1053ee9751dcd450776f7370c6f
MANIFEST DIGEST      sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6
```

And the second Run, whose first Step made no call:

```
ENTRY                01a070f7-52ac-7a19-9e63-2c81f4b0da95
PROCEDURE            form-cluster
CAUSE                manual
EXECUTOR             local
ACTOR                igor
HOST                 thinkpad
STARTED              2026-09-05T09:47:31.984Z
OUTCOME              completed
ENDED                2026-09-05T09:47:33.284Z
HYPER                0.0.3-alpha
PROCEDURE REVISION   ed503f1599a9f02ecd4e64677fc4c3f9ff17b52c
REPOSITORY REVISION  e81f88383d44df5aee872e299cdf1a3ab84dac38

STEP 1  create-control-plane · mutate · skipped-as-already-recorded
DEFINITION           cluster-machines
OPERATION            create_control_plane
PROVIDER             gce
TARGET               gce-prod
STARTED              2026-09-05T09:47:32.184Z
ENDED                2026-09-05T09:47:32.284Z
RECORDS              cp-1 — unchanged since run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47
DEFINITION REVISION  65968929dccfefcdff7457f36f1e3d5b1bbcbf13
MANIFEST DIGEST      sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84

STEP 2  create-workers · mutate · ran
DEFINITION           cluster-machines
OPERATION            create_worker
PROVIDER             gce
TARGET               gce-prod
STARTED              2026-09-05T09:47:32.384Z
ENDED                2026-09-05T09:47:32.584Z
RECORDS              node-1 · node-2 · node-3 · node-4
SELECTOR             values · node-1 · node-2 · node-3 · node-4
EXPANDED TO          node-1 · node-2 · node-3 · node-4
DEFINITION REVISION  65968929dccfefcdff7457f36f1e3d5b1bbcbf13
MANIFEST DIGEST      sha256:90d7585843ae50ee472ffae7ce26f4cc01ca83dbbe0699508159282a8994dc84

STEP 3  observe-cluster-membership · read · ran
DEFINITION           cluster-membership
OPERATION            list_nodes
PROVIDER             kubernetes
TARGET               cluster-api
STARTED              2026-09-05T09:47:32.684Z
ENDED                2026-09-05T09:47:33.184Z
RECORDS              cp-1 · node-1 · node-2 · node-3
DEFINITION REVISION  2e271a684ae9f1053ee9751dcd450776f7370c6f
MANIFEST DIGEST      sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6
```

The `RECORDS` cell reading `cp-1 — unchanged since run 01a06b8a-feac-7d31-b4a2-6f0c19e85d47` is
ADR-0055 read back: the Step file holds a digest and no members, so `show` resolves the set from the
Run that last carried one and **names that Run** rather than rendering the members bare.

**There is no `BOUND` line under either Step 2**, and `show` has one to render — three cases in the
suite's own corpus render it. Every one of those reads a Journal a test wrote by hand. That is finding
**#1**.

### `hyper records`

```
the record is the hyper-store branch of this repository — never checked out, and it travels with a clone

TARGET       DEFINITION          RECORD  ORDINAL  RUN             STEP  REHEARSAL  KIND         TOMBSTONE  ORPHANED  SECRETS  HYPER
cluster-api  cluster-membership  cp-1    1        01a06b8a-feac…  3                observation                                0.0.3-alpha
cluster-api  cluster-membership  node-1  1        01a070f7-52ac…  3                observation                                0.0.3-alpha
cluster-api  cluster-membership  node-2  1        01a070f7-52ac…  3                observation                                0.0.3-alpha
cluster-api  cluster-membership  node-3  1        01a070f7-52ac…  3                observation                                0.0.3-alpha
gce-prod     cluster-machines    cp-1    1        01a06b8a-feac…  1                asset                                      0.0.3-alpha
gce-prod     cluster-machines    node-1  1        01a06b8a-feac…  2                asset                                      0.0.3-alpha
gce-prod     cluster-machines    node-2  1        01a06b8a-feac…  2                asset                                      0.0.3-alpha
gce-prod     cluster-machines    node-3  1        01a06b8a-feac…  2                asset                                      0.0.3-alpha
gce-prod     cluster-machines    node-4  1        01a070f7-52ac…  2                asset                                      0.0.3-alpha
```

Nine Records, nine series, every ordinal `1`. `cp-1` appears twice under two Targets, which is the
Definition segment keeping two series apart. The `cluster-api` Observation of `cp-1` is still the
**first** Run's version, a Run later — and it stays that version through the third Run too.

### `hyper runs`

```
the record is the hyper-store branch of this repository — never checked out, and it travels with a clone

RUN             STARTED                   TRIGGER        OUTCOME    CONTESTED  PROCEDURE     TARGETS                 HYPER
01a070f7-52ac…  2026-09-05T09:47:31.984Z  igor@thinkpad  completed             form-cluster  cluster-api · gce-prod  0.0.3-alpha
01a06b8a-feac…  2026-09-04T08:31:06.512Z  igor@thinkpad  completed             form-cluster  cluster-api · gce-prod  0.0.3-alpha
```

### `hyper review procedures/form-cluster.yaml`

The whole Procedure is rendered verbatim with a gutter beside it — the file as §1 holds it, plus the
`node-4` the second Run's edit added — so only its head is worth repeating here:

```
  PROCEDURE            │  procedures/form-cluster.yaml            ed503f1 → working tree
  ─────────────────────┼────────────────────────────────────────────────────────────────
                       │   kind: procedure
                       │   procedure: form-cluster
  envelope ✓           │   targets: [gce-prod, cluster-api]
                       │   steps:
                       │     # the control plane is created already configured: kubeadm runs from cloud-init,
                       │     # and what it prints is published to Secret Manager, never back to hyper
  mutate  gce-prod     │     - id: create-control-plane
                       │       definition: cluster-machines
                       │       operation: create_control_plane
                       │       target: gce-prod
  ...
```

and its foot:

```
  AUTHORITY   assembled from definitions/ and targets/
  DEFINITION          TARGET       DEFINITION KINDS  TARGET KINDS  EFFECTIVE  DESTROY OPS
  cluster-membership  cluster-api  read              read          r          —
  cluster-machines    gce-prod     mutate            mutate        m          —

  FLAGS   index into the gutter above — no flag states anything the gutter does not
  ENVELOPE  line 3  ok  no step reaches a target outside [gce-prod, cluster-api]
```

**`ed503f1 → working tree` is the range**, opened against the revision the last Run of this Procedure
read. Before the first Run the same header read `no baseline — form-cluster has not run`, which is the
absence named rather than left blank.

**The Steps draw no flags.** `FLAGS` holds one entry, `ENVELOPE`, and it is not a warning — it is the
positive statement that no Step reaches a Target outside the Procedure's declared set. A Procedure that
creates four machines in a production project draws nothing else, and that is correct rather than
lenient: `DESTROY` needs a `destroy` Operation, `OPAQUE` needs the `shell` Capability, and `UNBOUNDED`
needs a `mutate` Step with no declared Bound. This repository grants no `shell`, ends nothing, and
declares a Bound on both effectful Steps — so the review's silence is three separate facts about the
artefacts, each earned.

The `AUTHORITY` table is the whole of the authority in one place: two Definitions, two Targets, one
effective Kind each, and no `destroy` Operations named anywhere.

---

## 8. The join secret's path

The scenario's whole reason for choosing Google Compute Engine is in this section: the join command
crosses **machine to machine**, and no part of it is ever in `hyper`'s hands.

Its path, in order:

1. the control plane's cloud-init runs `kubeadm token create --print-join-command` and writes the
   result to `/run/join` on that machine;
2. `gcloud secrets versions add kubeadm-join --data-file=/run/join` publishes it, authenticated by the
   **instance service account** the create request attached — an identity the machine already carries,
   with no credential anywhere in the request that made it;
3. `shred -u /run/join` removes the copy on disk;
4. each worker's cloud-init runs `gcloud secrets versions access latest --secret=kubeadm-join`,
   authenticated by the same kind of identity, runs the command it got, and shreds it.

What `hyper` holds is **the instruction to fetch a secret**, in a `runcmd` line of a request body, and
never the secret. The Step's arguments hold it; the Record holds the Manifest's four projected fields;
neither is the token, and no Step ever reaches inside a running machine to see one.

That claim is checkable rather than assertable, and the Store answers it:

```
$ # every string that could be a credential, counted over the whole Store branch
"ya29.a0AfB_example-access-token"                        0 occurrences in the Store
"eyJhbGciOiJSUzI1NiIsImtpZCI6ImV4YW1wbGUifQ.observer"    0 occurrences in the Store
"Authorization"                                          0 occurrences in the Store
"Bearer"                                                 0 occurrences in the Store
"kubeadm-join"                                           0 occurrences in the Store
"cluster-observer"                                       0 occurrences in the Store
"kubeadm token"                                          0 occurrences in the Store
```

The first two are the bearer tokens the two Targets resolved out of the environment — they went to the
wire and nowhere else. The last three are the *names* of secrets, which appear in the Procedure because
a machine is being told where to look, and appear nowhere in the record because the record holds what a
call returned rather than what a Step's arguments said.

**Nothing here needed a Secret sink.** `--secret-out` exists for an Operation that declares
`secret:` output — a value `hyper` itself received and must hand somewhere
([ADR-0148](../adr/0148-a-secret-sink-is-a-directory-hyper-makes-and-one-file-holds-one-value.md)).
No Operation here declares one, because no secret ever reaches `hyper` at all. That is the
difference between *`hyper` holds a secret carefully* and *`hyper` never holds it*, and it is the
property the scenario was built to reach.  **The cost is stated where part one states it**: without
a secret store this Procedure cannot form a
cluster. `hyper` supplies no place for a value to cross between two machines, and the model does not
grow one — the store is the operator's, and the dependency is real.

---

## 9. The cluster's own API, reached with `SSL_CERT_FILE`

The control plane signs its own API server certificate, so the paired `read` verifies against a root no
public store holds. There is no position for that root in any artefact, which is a twelfth negative
control beside §1's eleven:

```
$ printf 'ca: /etc/ssl/cluster-ca.pem\n' >> targets/cluster-api.yaml
$ hyper check --repo-dir cluster-repo
FILE                      LINE  FIELD  ERROR_CODE   MESSAGE
targets/cluster-api.yaml  9     ca     unknown-key  "ca" is not a key the schema at this position admits
$ echo $?
1
```

— and the route is the environment instead, which is
[ADR-0105](../adr/0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)'s
decisive property: **the trust is unreachable from where the agent writes**. An agent authoring
artefacts cannot add a root, because there is no key to add one with; an operator can, because
`SSL_CERT_FILE` is a variable on the process.

`hyper` reaches it by owning no TLS configuration at all. `cmd/hyper` wires
`new(tls.Dialer).DialContext` as the process's dialer (`cmd/hyper/main.go:81`); `internal/capability`
wires that dialer as `http.Transport.DialTLSContext` and holds no `tls.Config` of its own; so the roots
are Go's system pool, and Go's system pool reads `SSL_CERT_FILE`.

**The Runs of §3–§6 did not drive that**, and the reason is §2's fourth fixture: the harness's dialer
carries its own root pool, so their `read` verified against a pool a test built rather than against a
file an operator named. So the route is driven here instead, by the shipped binary, against a stand-in
for the cluster's own API — and driving it turned up the reason it could be driven at all.

### The stand-in, and the port that turned out to be writable

A `hosts:` grant **may carry a port**. This exercise believed otherwise while it authored §1, and the
repository's own acceptance harness has depended on the opposite all along:
`scripts/acceptance/lookout/fixture.sh` writes `hosts: [localhost:$port]` into a Target declaration
for a server on an ephemeral port. `check` agrees:

```
$ sed -i 's/hosts: \[k8s.hyper-example.dev\]/hosts: [k8s.hyper-example.dev:6443]/' targets/cluster-api.yaml
$ hyper check --repo-dir cluster-repo
checked 9 artefacts: no problems found
```

So the whole route is reachable. The stand-in is `providers/kubernetes.yaml` and
`definitions/cluster-membership.yaml` **exactly as §1 holds them**, a Procedure of the one `read` Step,
and a Target declaration granting the local port, against a TLS server holding a certificate signed by
the cluster CA and answering `/api/v1/nodes` with two nodes:

```yaml
kind: target-declaration
target: cluster-api
class: kubernetes
kinds: [read]
capabilities: [http]
hosts: [localhost:43903]
auth:
  token: {env: CLUSTER_OBSERVER_TOKEN}
```

Two Runs of the shipped binary, differing in one variable:

```
$ CLUSTER_OBSERVER_TOKEN=observer-token-example hyper run observe-cluster
run 01a0819b-9005-789b-843f-e93a9e0b695f
step 1/1 observe-cluster-membership
hyper run: step observe-cluster-membership: the collection path $.body.items did not resolve against
what came back, so hyper cannot tell a collection that was empty from a path that was wrong
STEP  ID                          KIND  DISPOSITION  RECORDS
1     observe-cluster-membership  read  ran          0 of 1

failed · exit 1 · run 01a0819b-9005-789b-843f-e93a9e0b695f

$ SSL_CERT_FILE=$PWD/ca.pem CLUSTER_OBSERVER_TOKEN=observer-token-example hyper run observe-cluster
run 01a0819b-9047-7864-ba9f-3a28e5453ac9
step 1/1 observe-cluster-membership
STEP  ID                          KIND  DISPOSITION  RECORDS
1     observe-cluster-membership  read  ran          2

completed · exit 0 · run 01a0819b-9047-7864-ba9f-3a28e5453ac9
```

The server saw both of them:

```
TLS handshake error from 127.0.0.1:40988: remote error: tls: bad certificate
GET /api/v1/nodes host=localhost:43903 auth=Bearer observer-token-example
```

Three things are on those two lines. The route **works** — a Run against an endpoint no public root
signs for completes when the environment names the root, and fails when it does not. The **port from
the grant reached the wire**, in the `Host` header, which is the finding above driven rather than
inferred. And the **credential reached the wire and nothing else**: `Bearer observer-token-example` is
the Manifest's scheme parameters over the Target's `env:` slot, composed at the request and held
nowhere — the Store this Run wrote holds no `Authorization`, no `Bearer` and no token, exactly as §8's
count says of the other three Runs.

The Observation it wrote is §1's Manifest projecting §1's fields, under a Procedure and a repository of
its own:

```json
{
  "definition": "cluster-membership",
  "fields": {
    "created": "2026-09-04T08:31:41Z",
    "kubelet_version": "v1.33.4",
    "name": "cp-1",
    "os_image": "Ubuntu 24.04.3 LTS"
  },
  "name": "cp-1",
  "operation": "list_nodes",
  "provenance": {
    "definition_revision": "2e271a684ae9f1053ee9751dcd450776f7370c6f",
    "hyper_version": "0.0.3-alpha",
    "manifest_digest": "sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6",
    "procedure_revision": "68eff1a2dc3c3a673d5d24608c57ecc8b7ad8be5",
    "repo_revision": "9baca3b19487269c014ee32b6cbddcaa11a246b4"
  },
  "record_type": "observation",
  "run_id": "01a0819b-9047-7864-ba9f-3a28e5453ac9",
  "schema_version": 1,
  "step": 1,
  "target": "cluster-api",
  "written_at": "2026-09-08T15:20:51.040Z"
}
```

`definition_revision` and `manifest_digest` are the ones §1 and §4 carry, the two files being
byte-identical to §1's; `procedure_revision` and `repo_revision` are this small repository's own.

### What the Run that could not verify recorded

This is the half worth reading twice, because it is what an operator who forgets the variable will be
looking at:

```json
{
  "definition": "cluster-membership",
  "disposition": "ran",
  "ended_at": "2026-09-08T15:20:50.976Z",
  "id": "observe-cluster-membership",
  "identities": {
    "digest": "sha256:37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570",
    "members": []
  },
  "kind": "read",
  "operation": "list_nodes",
  "projection_failed_path": "$.body.items",
  "provenance": {
    "definition_revision": "2e271a684ae9f1053ee9751dcd450776f7370c6f",
    "manifest_digest": "sha256:6235810ce0292606bb292512d2545b5b884945e8c96065b9a9122b39fa4d91f6"
  },
  "provider": "kubernetes",
  "schema_version": 1,
  "started_at": "2026-09-08T15:20:50.965Z",
  "step": 1,
  "target": "cluster-api"
}
```

**Nothing in it says the certificate did not verify.** The Disposition is *ran*; the identity set is
empty and its digest is the digest of an empty set; the one thing that says anything is
`projection_failed_path: "$.body.items"`, and the page says the same in a sentence about a JSON path.
That is each rule behaving as written — a `read` records the answer that came back, silence included
(ADR-0050); `answered` is written on no `read`; and no surface shows the response a projection failed
against (§7, ADR-0017) — and the sum of them is a Journal entry in which a trust failure and a
mistyped collection path are the same entry. It is finding **#15**.

A Manifest projecting one Record rather than a series would have recorded the silence instead: the
collection path is what fails first, and it fails before any field can carry a `status` that has gone
quiet. So the paired `read` — a `series` by construction — is the shape least able to say what went
wrong.

### The honest limit, measured

ADR-0105 already writes the cost of that route down, and writes it in **two halves**: *"on a machine
whose roots live in a hashed directory the fixture's root is **added** and public hosts still verify …
on a machine whose roots live only in a bundle file it is **substituted**, and for the length of that
run the fixture's root is the only one."* Both halves reproduce here, which is worth doing because the
half that travels is the second one — the ticket that asked for this exercise restates the limit
without its condition, and so would anything written from that restatement.

Go's system pool is assembled from a **file list** and a **directory list**. `SSL_CERT_FILE` replaces
the file list; `SSL_CERT_DIR` replaces the directory list; neither replaces the other, and the
directories are walked whatever the file variable says. Counted on the machine this exercise ran on,
with `roots.go` being six lines that call `x509.SystemCertPool()` and print how many subjects it holds:

```
$ go run ./roots.go
SSL_CERT_FILE="" SSL_CERT_DIR="" subjects=122
$ SSL_CERT_FILE=$PWD/ca.pem go run ./roots.go
SSL_CERT_FILE="/…/ca.pem" SSL_CERT_DIR="" subjects=123
$ SSL_CERT_FILE=$PWD/ca.pem SSL_CERT_DIR=$PWD/emptydir go run ./roots.go
SSL_CERT_FILE="/…/ca.pem" SSL_CERT_DIR="/…/emptydir" subjects=1
```

122 roots with nothing set; **123** with the cluster root named, the distribution's hashed directory
`/etc/ssl/certs` still supplying the other 122; and **1** when the directory list is pointed at an
empty directory, which is the substitution the ADR describes, isolated.

So the limit bites on a machine whose roots are a bundle **and nothing else**, and does not bite on the
Debian-shaped machine this ran on, which ships the hashed directory beside the bundle. ADR-0105 is
right as written; what this exercise found is that the sentence loses its condition the moment it is
paraphrased, which is finding **#9** and a warning to whatever states this limit next.

### What the stand-in cost, and what it did not

Two `hyper.yaml` keys and a `hyper store init`, and nothing else: the Manifest and the Definition are
§1's bytes, the Procedure is three lines, and the Target declaration differs from §1's in one host. No
artefact gained a key, no closed set moved, and nothing here needed a `ca:`, a flag, or a switch.

What it does not show is the four-Step Procedure carrying the variable, because the machines are
created against a cloud API that is not there. That is the shape of the thing: **the trust route and
the Run that needs it are separable**, and separating them is what makes the route checkable at all.

---

## 10. The findings

Everything above is a demonstration; this section is the result. Fifteen findings, numbered in the
order the document raises them and grouped below by what each one is — each stated as a finding rather
than as a complaint, and each of them a place the corpus did not decide the question, decided it in two
places differently, or stated something the binary does not do. **#1** and **#10** are the two that
cost something: one is a defect nothing can catch, and the other is a silence this exercise read the
wrong way and paid for in an artefact.

| # | What it is | What it takes to settle |
| --- | --- | --- |
| **1** | A Run never records a Step's Bound | a defect; one line of code and a golden that could catch it |
| **2** | The Step table renders a Disposition's wire spelling | a decision: which of the two spellings §12 fixes is the page's |
| **3** | The corpus prints the Run page indented and the binary does not | an edit to §8's examples, or a change to the page |
| **4** | `THE CODE MOVED` renders a non-scalar fact in a form nothing states | §8 gains what the binary already does |
| **5** | The canonical bytes of an identity digest | settled here by driving; §7 could carry the worked example |
| **6** | `show`'s human layout is stated nowhere | §8 gains the block this exercise drove |
| **7** | An empty change table renders no column-header row | settled here by driving |
| **8** | A paired `read` cannot see what the Steps beside it just did | a sentence in §13; there is nothing to build |
| **9** | `SSL_CERT_FILE`'s honest limit loses its condition when restated | whatever states it next carries both halves |
| **10** | A host in a `hosts:` grant may carry a port; the corpus is silent | one sentence in §3, either way |
| **11** | A Run of three Steps can write no Record version at all | a sentence where the Journal's shape is described |
| **12** | A Run reads the clock many times, and its commits carry one instant | a sentence in §7 |
| **13** | Two Steps of one Run can carry one identity digest | a sentence where the digest is defined |
| **14** | An Asset written against an asynchronous create is about the job | §13's new subsection, which is written from this |
| **15** | A `read` that could not verify the host records a JSON path | a decision about what a `read` may record of a transport |

### The three that are defects

**1. A Run never records a Step's Bound, and the specification, the Store's own schema and one
rendering all assume it does.**
This is the largest thing the exercise found. §7 says a Disposition carries *"the selector it resolved
together with what that selector expanded to **and the Bound it was counted against**"*; §7's own worked
Step file carries `"bound": 5`; `internal/store`'s `Selector` holds a `Bound` member and writes it where
it is non-zero; §9 lists the Bound among the things `show --expansion` carries; and `hyper show
--expansion` renders a `BOUND` line from it. `internal/run/step.go:140`
builds the value that gets written —

```go
file.Selector = store.Selector{Declared: expanded.Selector.Declared, ExpandedTo: expanded.names()}
```

— from two of the three members, so **no Run has ever written a Bound into a Journal**. Every Step file
in §4 and §5 above is a Step declaring one and recording none.

Why no golden catches it: the three `show` cases that render a `BOUND` line all read Journals **seeded
by hand**, and the run corpus's own Step files are compared against goldens generated from the same code
that omits it. A fixture that writes the field and a binary that does not are consistent with each other
everywhere except where a real Run meets a real `show`, which is what this exercise did.

What is lost is small and specific: the Journal cannot answer *what was this Expansion counted against*,
so an entry read after the artefact has moved cannot say whether a Bound was widened before or after the
Run. `changes` still reports a Bound moving, because its `Bounds` class reads the **artefact** at two
revisions rather than the Journal (§8) — which is why the surface that would most obviously have shown
the absence does not.

**2. The Step table renders the wire spelling of a Disposition where §12 fixes a phrase beside it.**
§12 states each Disposition as a name with its wire spelling: *"**skipped as already recorded** —
`skipped-as-already-recorded`"*. §8's prose uses the name throughout. The page renders the wire
spelling:

```
STEP  ID                          KIND    DISPOSITION                  RECORDS
1     create-control-plane        mutate  skipped-as-already-recorded  1
```

Every golden in the suite agrees with the binary, so this is not a regression — it is the corpus
deciding one question in two places. Either the page renders the name, in which case the column widens
by nothing and four corpora move, or `--json` and the page share one spelling and §12's two-column form
is a wire spelling with a gloss rather than a rendering. `CONTRIBUTING.md` decides which way a
disagreement points — *where the code and `docs/spec/` disagree, the spec is right and the code is the
defect* — so the burden is on the page. It is worth deciding rather than inheriting.

**3. The corpus prints the Run page with a two-space indent and the binary prints it flush left.**
§8's Step table and terminal line are printed indented; so is its `changes` block, and the `changes`
page **is** indented. The Run page is not:

```
STEP  ID      KIND     DISPOSITION  RECORDS      ← the binary
  STEP  ID      KIND     DISPOSITION  RECORDS    ← §8
```

Of the six surfaces this exercise drove, the corpus prints three — `run`, `changes` and `review` — and
one of the three does not match. A reader building a fixture or an expectation from §8's printed bytes
gets it wrong for exactly one surface, which is the failure mode a worked example exists to prevent.
By the same rule as **#2** the spec is the side that is right, which here would mean indenting the page;
if the indent in §8's blocks is typography rather than bytes, that is the thing to say, once, where the
surfaces are stated.

### The four the corpus leaves open, and this exercise answers

**4. `THE CODE MOVED` renders a non-scalar fact on two lines, and nothing states it.** The Tailscale
exercise raised this as its finding **#5**: the `cadence` row's stacked form is specified in full and its
four siblings — selector, Target set, Operation set, `destroy` Operations — are not. The binary's answer,
driven here:

```
  procedure form-cluster  step create-workers · over   values                    values
                                                       node-1 · node-2 · node-3  node-1 · node-2 · node-3 · node-4
```

The class goes on the fact's own line and the members stack beneath, aligned under `FROM` and `TO`. A
twenty-member `values:` list still has no stated behaviour, and neither does a five-conjunct `assets:`
predicate; but the shape exists and §8 can adopt it rather than invent a second one.

**5. The canonical bytes an identity digest is taken over.** The Tailscale exercise's finding **#4**
read §7's *"the canonical JSON encoding of the sorted array, trailing LF included"* as a two-space
indent, one element per line and no trailing comma, and noted that a different reading changes every
digest in every Journal ever written — in a Store that is append-only, which makes the question
irreversible rather than merely open. That reading is the implementation's. All four digests this
exercise produced reproduce from the names alone:

```
$ printf '[
  "node-1",
  "node-2",
  "node-3",
  "node-4"
]
' | sha256sum
b23bc5b47f21666498ed8c277a12af0ae7655c0fe15cdd2d16624e88e14054ef  -
```

§7 can close it with one worked example, and the question stops being a risk the moment it carries one.

**6. `show`'s human layout is stated nowhere, and the corpus now holds one invented rendering of it.**
The Tailscale exercise recorded that §9 says what `show` carries — the Dispositions, the identities,
the Pattern account, and under `--expansion` the selector, `expanded_to` and the Bound — while §8
states the verbatim form of five other renderings and not this one. It drew its own block — a
two-space-indented `RUN` header with `STEP n` sections and lower-case labels. The binary draws a
flush-left labelled block with upper-case labels and a different label set (`ENTRY`, not `RUN`;
`RECORDS`, not `records`). Both documents say the layout is unspecified; one of them now shows a
layout that is not the tool's. §8 should state one, and this exercise supplies the bytes. 
**7. An empty change table renders its header and its count and no column-header row.** The Tailscale
exercise's finding **#14** left this open, reading §8's *"header and count"* as excluding the column
header. Driven above in §6: that reading is the binary's.

### The four that are consequences worth naming

**8. A paired `read` cannot see what the Steps beside it just did, and the practice's value lands one
Run later.** In the first Run the three workers were created and the `read` beside them found one node;
in the second, the workers were there and `node-4` was not. Nothing in the model can close that gap:
a Requirement's predicate reads Records and halts the Run rather than waiting, a polling Pattern lives
**inside** one Operation and cannot poll a different system, and `skip-if-recorded` trusts the Record
over the world. So the pairing is real and its evidence is always about the **previous** Run's effects.
That is not a flaw — it is what an asynchronous world does to a procedural tool, and the Comparison
renders it honestly — but a reader told *pair a `read` with your effectful Step so the Comparison has
something to render* will expect it to render something about the Step it is paired with.

**11. A Run of three Steps can complete having written five Journal files and no Record version at
all.** The third Run: two Steps skipped every member and the `read` observed exactly what it observed
before, so identical bytes minted nothing. The conclusions are on the Step files as identity sets, and
`records` is unchanged. This is the steady state of an immutable fleet, and it is the state a reader is
most likely to misread as *the Run did not do anything*.

**12. A Run reads the clock many times, and every commit it writes carries the Run's start instant.**
Fifteen reads for each of the first two Runs here and fourteen for the third, which wrote no Record
version. The Journal's `started_at`, `ended_at` and `written_at` advance through a Run accordingly;
§7's rule that *the instant on this Run's `run.json`, fixed at its start* is shared by every Step is
about **relative predicates**, not about those members. The Store's own commits take the start
instant, truncated to whole seconds. Both facts are correct and neither is stated where a reader of
the Journal would find them.

**13. Two Steps of one Run can carry one identity digest and mean two different things.** Step 1 and
Step 3 of the first Run both concluded about `{cp-1}` — one an Asset under `gce-prod`, one an
Observation under `cluster-api` — and both carry
`sha256:f7b3085a922910ffbd9a68980b0e463b2edf274fb825f277ffb3c3749fcd84c1`. The digest is over the sorted
names and nothing else. This is safe because ADR-0055's *unchanged since* walks the history of that
Step, but a reader who takes a digest for an identifier of a set of Records will be wrong.

### The four that are about a limit rather than about a mechanism

**9. `SSL_CERT_FILE`'s honest limit is conditional, and the condition is what falls off in the
retelling.** ADR-0105 states both halves — added where a machine's roots are a hashed directory,
substituted where they are only a bundle — and both reproduce: 122 roots bare, 123 with the cluster
root named, 1 with the directory list emptied. The ticket that commissioned this exercise restates the
limit as *"on a machine whose roots live in a bundle file the root is substituted rather than added, so
public hosts stop verifying for that Run"*, which is the second half alone; a §13 subsection written
from that restatement would tell an operator on a Debian-shaped machine to expect a breakage that
cannot happen there, and an operator on a minimal image nothing at all about what saves them. The
finding is about the sentence's carriage, not about the ADR.

**10. A host in a `hosts:` grant may carry a port, and nothing in the corpus says so in either
direction.** §3 calls `hosts:` *a list of hosts*; ADR-0082 fixes the scheme and says nothing about a
port; §4's checks do not look. `check` accepts `hosts: [k8s.hyper-example.dev:6443]`, the port reaches
the wire in the `Host` header, and the repository's **own acceptance harness has depended on this all
along** — `scripts/acceptance/lookout/fixture.sh` writes `hosts: [localhost:$port]` for a server on an
ephemeral port. This exercise read the silence the other way and paid for it in §1: a cluster told to
bind 443 to meet a constraint that was not there. A silence two readers read oppositely is a sentence
missing, and the sentence is cheap either way — *a host may carry a port, and `{from-target}` carries
it* is one line in §3, and so is its opposite plus a `check` that enforces it.

**14. An Asset written against an asynchronous create is accountable for the job, not for the machine.**
Every field on `cp-1`'s Asset comes off a `compute#operation` — the name of a job, its `RUNNING` status,
the link to the instance it will make, and when it started. The Record is true and says nothing about
whether a machine exists. It is the same accountability limit an Opaque Operation has, reached from a
completely different direction: not *the command ran and the service is unknown*, but *the call was
accepted and the machine is unknown*. §13's new subsection is written from this, and it should name both
shapes, because an author who avoids `shell` for the reason that section gives will land here.

**15. A `read` that could not verify the host records a JSON path and nothing about trust.** Driven in
§9: the same Run, twice, differing in `SSL_CERT_FILE`. The one that could not verify writes a Step file
whose Disposition is *ran*, whose identity set is empty, and whose only account of what happened is
`projection_failed_path: "$.body.items"` — the same entry a mistyped collection path would write. Each
rule is behaving as stated (ADR-0050, `answered` on no `read`, ADR-0017's *no surface shows the
response*), and the sum is that the Journal cannot tell a trust failure from an authoring error. An
operator's first move — read the entry — is the move that cannot help. What would help is not obvious
and is a decision rather than an edit: a `read` may not record a status the Manifest did not project
without ADR-0010's line moving.

### What this scenario never reached

Stated so that the coverage is not read as complete. No `destroy` and therefore no Tombstone, no
`bound-exceeded` and no halt; no Refusal and no non-`completed` outcome; no Cadence, so no projected
workflow, no job summary and no staleness reading; no nested Procedure, so no Step `path`; no
Requirement; no `when:` condition; no Pattern that did more than the trivial single call, so no Pattern
account anywhere; no Secret sink; no `shell` and nothing Opaque; no percent-encoded or over-long Record
name; no Compaction; and no Run on a runner, so every Trigger here is `manual`/`local`.

---

## 11. The freeze

The header states it; this section says what it covers, because part one stood unfrozen for a while and
somebody will reasonably ask what that changed.

**Everything below the header is frozen**: the artefacts, the Store files, the renderings, the
digests and the findings. A finding that gets fixed is fixed in the tree and recorded wherever the fix
lands — it is not struck out here, because this file's subject is what was true at `3825d73` and a
document edited to stay current is a document checked against nothing. The two exemptions are the two
edits that assert nothing about the corpus: a relative link that has gone stale, and a typographical
error that changes no fact.

Part one carried a *not yet frozen* marker while it was the only half there was, which is why the
freeze is part two's to own: the file that carries the rule is the file the rule has always been true
of.
