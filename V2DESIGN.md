# DeployGrid v2 Design

DeployGrid answers one question at a glance: **for every component we ship,
which version is running in each environment, and where?** The grid has
components as rows and environments as columns; each cell is a version with
enough context (cluster, namespace, health, links) to act on it.

This document evaluates the v1 implementation on `main` and the `try-v2`
experiment, and sets out the model, conventions and infrastructure for v2.
The CRD types described here are implemented in
`pkg/apis/deploygrid.activated.io/v1alpha1` on this branch; the rest is a plan.

---

## 1. Where we are

### v1 (`main`)

* Single Go service plus a React table. One endpoint: `GET /api/grid`.
* **Pull model.** The server holds a kubeconfig per cluster (`clusters:` in
  `config.yaml`), runs a reflector per cluster for Argo CD `Applications` and
  `apps/v1 Deployments`, and rebuilds the whole grid on every request.
* **Discovery by annotation.** An Application annotated with
  `deploygrid/name`, `deploygrid/environment` and `deploygrid/group` becomes a
  row cell; its `destination.server` is matched back to a configured cluster
  address to find the Deployments it manages (via
  `app.kubernetes.io/instance`), whose container image tags become child rows.
* Environments are a static list in config; the grid is system-agnostic.

It works for the kind fixture, and the reflector/store layering
(`pkg/repository`, `pkg/service/store.go`) is sound and reusable.

### `try-v2` (before this branch's changes)

* Started a CRD layer (`System`, `Metadata`, `MetadataView`) with wrangler
  codegen, but the types were still kubebuilder scaffolding (`Foo` fields).
* Started moving config from viper to `activatedio/cs`, half-way: `ctors.go`
  referenced symbols that no longer existed, `cmd/main` imported a deleted
  package. The branch did not compile.
* `V2DESIGN.md` listed REST paths and asked the key question: keep pulling
  from clusters, or have clusters push?

---

## 2. Evaluation

### 2.1 Model

| Finding | Why it matters |
|---|---|
| No notion of **System**. One grid for everything. | Can't serve several products/teams from one install; environment lists differ per product. |
| **Environment is a config list**, and a per-Application annotation. | Columns and cells can disagree; nothing owns the environment order or display name. |
| `Component.ComponentType` is a free string that conflates Group, chart and container rows. | UI special-cases `"Group"`; no stable identity for a row across environments. |
| Row identity is derived from `pathElement` (deployment name + container name). | Any naming difference between environments splits a row in two. |
| A cell is just `{version}`. | No cluster/namespace, no health, no desired-vs-actual, no links, no timestamps. |
| No history. | "When did QA move to 1.4?" and "what changed since yesterday?" are unanswerable. |
| Desired and actual are not distinguished. | Argo CD `targetRevision` is *desired*; the image tag is *actual*. Drift between them is the most useful signal and it's invisible. |
| `MetadataService`/`MetadataRenderer` interfaces exist with no implementation or model. | The "named configurations rendered through templates" idea needs a home in the model. |

### 2.2 Conventions

| Finding | Recommendation |
|---|---|
| Annotation prefix `deploygrid/` | Use the DNS-subdomain prefix `deploygrid.activated.io/` as Kubernetes convention expects; read the old keys for one release. |
| Everything is an annotation | Use **labels** for identity (`system`, `component`, `environment`) so collectors can use label selectors; annotations only for free-form display data. |
| Clusters identified by matching `destination.server` URL (`addressMap`) | Keep as one matcher, but make the Cluster a first-class resource with a list of addresses and also honour Argo CD `destination.name`. |
| Image version = `strings.Split(image, ":")[1]` | Breaks on `registry:5000/img` and `@sha256:` digests. Replaced with `ParseImageReference` (`pkg/repository/k8s/image.go`). |
| Chart version read only through Argo CD | Also read the Helm release Secret (`sh.helm.release.v1.*`) or `helm.sh/chart` label so plain Helm installs work without Argo CD. |
| Config keys via viper `mapstructure` tags, env via ad-hoc binding | `cs` with `DEPLOYGRID_*` overrides, now complete and tested (`pkg/config`). |

### 2.3 Infrastructure

| Finding | Status |
|---|---|
| Server needs a kubeconfig for every cluster, mounted from Secrets. | Root cause of the next three rows; v2 inverts this (see §4). |
| `accessor.go` forced `TLSClientConfig.Insecure = true` for every cluster ("TODO remove"). | Now opt-in per cluster via `insecureSkipTLSVerify`. |
| Runtime image is Debian + `awscli` (for EKS auth) ≈ 300 MB. | Push model removes the need; target distroless. |
| ClusterRole grants `get/list/watch` on `*/*`. | Narrowed to the resources observed plus the deploygrid API group. |
| CORS allows every origin **with credentials**. | Serve UI and API from one origin (nginx proxies `/api`), drop permissive CORS. |
| `release.yaml`/`test.yaml`/`lint.yaml` pointed at `./api`, which no longer exists after the reorg. | Fixed. |
| Chart: UI container used the API image tag; `crds.enabled`/`crds.keep` referenced but not in `values.yaml` (template fails to render). | Fixed. |
| Reflector goroutines used `break` inside `select`, so they never exit on context cancel and the retry loop spins. | Fixed in `resource.go`. |
| `go test ./...` requires three kind clusters (no build tag separates unit from integration tests). | Add `//go:build integration` or use the fake dynamic client; see §7. |
| `.golangci.yml` is v2 format while the Makefile pins golangci-lint v1.61; lint config references crossplane import groups. | Align on v2 and the project prefix. |
| UI `dist/` is committed and `VITE_API_URL` is baked at build time. | Build in CI only; use a relative `/api` base. |

---

## 3. v2 model

### 3.1 Vocabulary

| Term | Meaning | Where it lives |
|---|---|---|
| **System** | One grid: a product or platform boundary. Declares ordered **environments** (columns) and **groups** (row sections). | `System` CR |
| **Component** | One row: a deployable unit with an identity that is stable across environments. Can be declared by a person or **discovered** from observations and then enriched. | `Component` CR (spec = declaration, status = current cells) |
| **Cell** | (system, component, environment) → actual version, desired version, cluster, namespace, health, hosts, timestamps. | `Component.status.environments[]` plus the API |
| **Cluster** | A registered Kubernetes cluster, its addresses, how it is collected, and how its namespaces map to environments. | `Cluster` CR |
| **Observation** | A collector's report of what is running: a normalised list of artifacts (Argo CD application, Helm release, workload container, ingress host) with labels and parent links. | Server memory; pushed over the API; not a CR |
| **Configuration** | A named document of values for a System, optionally scoped to one environment; environment documents deep-merge over the system-wide one. | `Configuration` CR |
| **ConfigurationView** | A Go template that renders a Configuration (and the grid cells) for an environment, e.g. to emit a values file or hosts list. | `ConfigurationView` CR |

Metadata/MetadataView from the earlier draft are renamed Configuration /
ConfigurationView to match the "named configurations" wording and the REST
paths in the original notes.

### 3.2 Why CRDs for the declared half

* `kubectl get components -n deploygrid` shows the grid in a terminal with
  print columns; the status *is* the grid, persisted.
* Systems, Components and Clusters are GitOps-able and reviewable.
* No database to run. Observations are re-sent in full by collectors on
  connect (same `Replace` semantics the reflector already has), so the server
  is stateless apart from the CRs it owns.

Observed data stays out of CRDs: it is high-churn and large. The server keeps
it in memory (the existing `Store`), and writes only the *summary* into
`Component.status`.

### 3.3 Resolution rules

**Component identity** for an observed artifact, first match wins:

1. Label `deploygrid.activated.io/component` (or legacy annotation `deploygrid/name`).
2. A declared `Component.spec.selector` whose criteria all match.
3. For a workload managed by a Helm release / Argo CD app, the identity of the parent.
4. Otherwise the artifact is unassigned; it is listed under *Unassigned* in the
   API so operators can see what they have not labelled, and a `Component` is
   created with `status.discovered: true` only when discovery is enabled for the System.

**Environment** for an observed artifact, first match wins:

1. Label/annotation `deploygrid.activated.io/environment`.
2. The owning Argo CD Application's environment (labels on the Application).
3. `Cluster.spec.namespaceRules` (glob on namespace).
4. `Cluster.spec.environment`.

**Version**, by component kind:

| Kind | Actual | Desired |
|---|---|---|
| `helm-chart` | Helm release Secret chart version, else `helm.sh/chart` label | Argo CD `source.targetRevision` |
| `container` | image tag, else short digest (`ParseImageReference`) | Argo CD rendered manifest (phase 3) |
| `argocd-application` | `status.sync.revision` | `spec.source.targetRevision` |

A cell is **drifted** when desired is known and differs from actual, and
**inconsistent** when the same component reports two actual versions in one
environment (two clusters, or two namespaces).

**Health** rolls up from the workload: Deployment `Available`/`Progressing`
conditions and Argo CD `status.health`, into `Healthy | Progressing | Degraded | Unknown`.

### 3.4 Labels and annotations

```
deploygrid.activated.io/system:       <System name>          label
deploygrid.activated.io/component:    <Component name>       label
deploygrid.activated.io/environment:  <environment name>     label
deploygrid.activated.io/group:        <group name>           annotation (display only)
deploygrid.activated.io/display-name: <text>                 annotation
```

Legacy `deploygrid/name`, `deploygrid/environment`, `deploygrid/group` are
read for one release and reported as a warning in the API.

---

## 4. Collection: push, with pull kept for compatibility

**Decision: collectors run in (or next to) each observed cluster and push
observations to the server.** The server never needs a kubeconfig for a
remote cluster.

Why:

* Network direction. Agents connect outward to the server; the server does
  not need routes, VPNs or public API endpoints into every cluster.
* Credentials. Each agent uses its own in-cluster service account with a
  narrow read-only role. No kubeconfig Secrets, no cloud CLI in the server
  image, no `Insecure: true`.
* Scale. Reflectors and caches live where the data is. The server receives
  deltas instead of maintaining N watches.
* Failure isolation. A down cluster is a stale `Cluster` heartbeat, not an
  error on the shared request path.

Pull is still supported for two cases: the cluster the server runs in
(`mode: local`) and development against kind (`mode: kubeconfig`). The
collector code is identical; only the client source and the sink differ.

### 4.1 One binary, three modes

```
deploygrid server     # API + UI, reads CRs from the control cluster, receives observations
deploygrid collector  # watches a cluster, pushes observations to --server
deploygrid all-in-one # server + local collector (single cluster, dev)
```

The collector is the existing `pkg/repository/k8s` reflector pipeline plus
new sources, feeding an **Observation sink**:

```go
type Observation struct {
    Cluster   string
    Snapshot  bool            // true: full replace; false: delta
    Artifacts []Artifact
    Removed   []ArtifactRef
    SentAt    time.Time
}

type Artifact struct {
    Ref         ArtifactRef           // kind, namespace, name
    Kind        ArtifactKind          // argocd-application | helm-release | deployment | statefulset | daemonset | ingress
    Labels      map[string]string
    Annotations map[string]string
    Parent      *ArtifactRef          // workload → helm release / application
    Versions    []Version             // containers, chart, revision
    Hosts       []string              // ingress
    Health      Health
    Destination *Destination          // for applications: server/name + namespace
}
```

Sinks: `InProcessSink` (all-in-one, local) and `HTTPSink` (bearer token from
`Cluster.spec.collection.tokenSecretRef`, gzip JSON, retry with backoff,
full snapshot on reconnect).

Sources, in order of delivery:

1. `apps/v1` Deployments, StatefulSets, DaemonSets (containers → versions, conditions → health).
2. `networking.k8s.io/v1` Ingresses (hosts, matched to workloads via Service selectors).
3. Argo CD Applications (desired version, destination, health).
4. Helm release Secrets (`type: helm.sh/release.v1`; chart name/version without Argo CD).

### 4.2 Server pipeline

```
observations  ──▶  per-cluster Store (Replace/Add/Modify/Delete)
                       │
Cluster CRs   ──▶  environment resolution ─┐
Component CRs ──▶  identity resolution   ──┼──▶  Grid index (system → component → env → cell)
System CRs    ──▶  columns/groups         ─┘          │
                                                      ├──▶ GET /api/systems/{s}/grid (read from index)
                                                      ├──▶ Component.status writer (debounced)
                                                      └──▶ History appender (version changes)
```

The index is recomputed incrementally per cluster event, not per HTTP
request. `Component.status` is written at most every few seconds per
component and only when a cell changes.

---

## 5. API v2

All under `/api`. `GET /api/grid` remains as the grid of the first System.

```
GET  /systems                                   list systems
GET  /systems/{s}                               environments, groups
GET  /systems/{s}/grid                          rows (grouped, nested by parent), columns, cells
GET  /systems/{s}/components                    declared + discovered components
GET  /systems/{s}/components/{c}                one row with all cells and links
GET  /systems/{s}/components/{c}/history        version changes, newest first (?environment=, ?since=)
GET  /systems/{s}/unassigned                    observed artifacts that matched no component
GET  /systems/{s}/configurations                names
GET  /systems/{s}/configurations/{name}         merged values (?environment=)
GET  /systems/{s}/views/{view}                  rendered view (?environment=) with the view's content type
GET  /clusters                                  registered clusters with heartbeat/connected
POST /observations                              collector push (bearer token identifies the Cluster)
GET  /healthz
```

Grid response shape (cells carry what the UI needs to colour and link):

```go
type Grid struct {
    System       SystemRef
    Environments []Environment            // ordered columns
    Groups       []GridGroup              // ordered sections
    Warnings     []string                 // e.g. legacy annotations seen, cluster disconnected
}
type GridGroup struct { Name, DisplayName string; Rows []GridRow }
type GridRow struct {
    Component   ComponentRef              // name, displayName, kind, discovered
    Cells       map[string]*Cell          // by environment name
    Children    []GridRow                 // spec.parent nesting
}
type Cell struct {
    Version, DesiredVersion string
    Drifted, Inconsistent   bool
    Health                  string
    Cluster, Namespace      string
    Hosts                   []string
    Links                   []Link        // rendered from Component.spec.links
    LastChange, LastSeen    time.Time
    Sources                 []ArtifactRef // what this cell was computed from
}
```

Mutating endpoints for Systems/Components are deliberately absent in v2:
`kubectl` and GitOps are the write path. If the UI later needs to enrich a
discovered component, add `PATCH /systems/{s}/components/{c}` that writes the
CR.

---

## 6. History

Append-only version changes per cell:

```
(system, component, environment, cluster, from, to, observedAt, source)
```

Storage: start with an in-memory ring per cell plus Kubernetes `Events` on the
`Component` (free audit trail, visible in `kubectl describe`). If retention
beyond the event TTL is needed, add a `History` interface with a SQLite
(`modernc.org/sqlite`, no CGO) implementation behind a PVC. Decide after
seeing real usage; the interface boundary is cheap, the database is not.

---

## 7. Infrastructure

**Packaging.** One Go module, one image (`deploygrid`), `distroless/static`
runtime, `CGO_ENABLED=0`. The UI is served by nginx which also proxies `/api`
to the server, so the browser sees one origin and CORS can be disabled. The
chart grows a `collector` sub-chart (DaemonSet-free single Deployment, read-only
ClusterRole, Secret with the server token) that can be installed into every
observed cluster with just `server.url` and `cluster.name`.

**Control cluster.** The chart installs the CRDs (`crds.enabled`, `crds.keep`)
and the server reads CRs from its release namespace (`control.enabled`,
`DEPLOYGRID_CONTROL_NAMESPACE`). Status updates need `update/patch` on
`deploygrid.activated.io` resources; the RBAC template now grants that.

**Configuration.** `cs` with YAML plus `DEPLOYGRID_*` environment overrides.
Sections: `logging`, `server`, `swagger`, `control`, `clusters` (v1
compatibility, removed when the collector ships). `pkg/config/config_test.go`
pins the behaviour.

**Development.** `make dev_kind` creates ops + two app clusters and applies
Argo CD CRDs, sample Applications/Deployments, the deploygrid CRDs and
`kind/systems.yaml`. `make serve` runs the server against them with the
control cluster on `kind-ops-cluster-1`.

**Tests.** Three tiers:

* Unit: pure functions (`ParseImageReference`, grid builder, resolution rules,
  config). Run everywhere.
* Component: reflector/store against `k8s.io/client-go/dynamic/fake` and
  wrangler fake clients. Run everywhere. Move `pkg/repository/k8s/*_test.go`
  here or tag them.
* Integration/e2e: kind, tagged `//go:build integration`, run in CI only.

**CI.** `test.yaml`/`lint.yaml`/`release.yaml` now target the repo root. Pin
golangci-lint v2 in the Makefile to match `.golangci.yml`, and replace the
crossplane import-group prefixes with `github.com/activatedio`.

---

## 8. Roadmap

**Phase 0 — this branch (done).**
CRDs for System, Component, Cluster, Configuration, ConfigurationView with
generated wrangler clients and chart templates; `cs` configuration with
defaults, env overrides and tests; control-cluster informers;
`GET /api/systems`, `GET /api/systems/{s}`, `GET /api/systems/{s}/grid`
(columns from the System, cells from the v1 builder); bug fixes (reflector
goroutine exit, image tag parsing, TLS hack opt-in, CI paths, chart rendering);
kind fixtures with sample CRs.

**Phase 1 — v2 grid from CRs + existing pull (done, 2026-10-01).**
`repository.Resource` is the normalised artifact (kind, namespace, parent,
versions, desired version, chart version, health, destination); the
Application and Deployment collectors fill it. `pkg/grid` is the pure
builder: resolution rules (§3.3), cells with desired/actual, drift,
inconsistency, health rollup, rendered links, parent nesting, group order,
discovered rows, unassigned list and legacy-annotation warnings, all covered
by fixture tests. `pkg/service` supplies a `Catalog` (custom resources, or a
"default" System synthesised from v1 config), the grid service, and a status
writer that persists cells into `Component.status` and counts into
`System.status` every 15 s when they change. Endpoints: `/systems/{s}/grid`,
`/components`, `/components/{c}`, `/unassigned`. The UI has a system picker
and renders groups, nested rows, drift ("wants x.y.z"), health colours,
links and artifact detail on hover. Configured `clusters:` are merged with
`Cluster` CRs by name, so the kubeconfig still comes from config while
environment mapping comes from the CR.

Not yet in phase 1: ingress hosts, Helm release Secrets, the per-request
rebuild (the grid is still computed on each GET; fine at current scale, the
index in §4.2 is the fix when it is not), and `/history`.

**Phase 2 — collector mode (done, 2026-10-01).**
`deploygrid collector` watches a cluster with the same repositories the
server uses and pushes `Observation`s to `POST /api/observations`: a
snapshot per kind on first sync and after any failure, coalesced deltas
otherwise, a heartbeat when idle, and a full resync when the server asks
for one (for example after it restarted). The server keeps every cluster's
state in a `SourceRegistry` that server-side watches and pushed
observations both feed. Tokens: a static `token` on an agent-mode entry in
`clusters:`, or, with a control cluster, the Secret referenced by
`Cluster.spec.collection.tokenSecretRef`; the server generates
`<cluster>-collector-token` for agent-mode Clusters that reference none.
The status writer sets the `Connected` condition and heartbeat fields on
every `Cluster`. Ingress hosts are collected and shown on cells.
`charts/deploygrid-collector` installs the collector with a read-only
ClusterRole and the token Secret. The e2e test feeds one of the three kind
clusters through an in-process collector authenticated with the generated
token.

Deferred from phase 2: the distroless image (the server image still carries
awscli for kubeconfig-mode EKS clusters; drop it once those move to
collectors) and Helm release Secrets as a source (the `helm.sh/chart` label
already carries the chart version; decoding release Secrets needs cluster-wide
Secret access, which the collector role deliberately does not have).

**Phase 3 — configurations, history, UI (done, 2026-10-01).**
`pkg/configuration` deep-merges a system-wide `Configuration` with the
environment document of the same logical name (`<name>` and
`<name>-<env>`) and renders `ConfigurationView` templates with a small
function set (`default`, `upper`, `join`, `toYaml`, `toJson`, `indent`,
...) over `.Values`, `.Components` (the environment's rows) and `.Grid`.
A view names the configuration it renders (`spec.configuration`, default:
its own name). Endpoints: `/configurations`, `/configurations/{name}
?environment=`, `/views`, `/views/{view}?environment=`. The reconciler
validates every view template and records `TemplateValid` on its status.

History: `pkg/history` keeps a bounded ring of version changes per cell,
diffed by the single reconciler loop every 15 s (with a 45 s warm-up after
start so cells appearing while watches sync are seeded, not reported).
Each change is also recorded as a Kubernetes Event (`VersionChanged`) on
the Component, which is the durable record (`kubectl describe component`).
Endpoints: `/systems/{s}/history` and `/components/{c}/history` with
`?environment=&since=&limit=`.

UI: component detail page (hash route `#/systems/{s}/components/{c}`) with
one card per environment (version, desired, health, cluster, hosts, links,
artifacts) and the recent-change list; an "only drifted, inconsistent or
degraded" filter on the grid.

---

## 9. Decisions to confirm

1. **Discovery default.** *Decided 2026-10-01:* auto-create. A resource that
   carries a component identity no `Component` declares becomes a `Component`
   resource with `status.discovered: true` (spec filled from what was
   observed: system, group, kind, display name) so it can be enriched in
   place. `System.spec.discovery.createComponents: false` turns this off for
   a system, in which case discovered rows stay in memory only. Resources
   with no identity at all are still listed under `/unassigned`.
2. **Cell per cluster or per environment.** *Decided 2026-10-01:* one cell
   per environment. When clusters disagree the cell is marked *inconsistent*
   and lists every cluster it was built from.
3. **History storage.** *Decided 2026-10-01:* Events first. Version changes
   are kept in an in-memory ring per cell and recorded as Kubernetes Events
   on the Component; a durable store is added only if the Event TTL proves
   too short in practice.
4. **UI composition.** *Decided 2026-10-01:* keep the separate nginx UI
   container and proxy `/api` through it; do not embed the UI in the Go binary.
