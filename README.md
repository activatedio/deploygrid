# DeployGrid

DeployGrid answers one question at a glance: **for every component we ship,
which version is running in each environment, and where?** Components are
rows, environments are columns, and each cell shows the running version, the
desired version, drift, health, cluster, namespace, hosts and links.

## How it works

```
 observed clusters                         control cluster
 ┌──────────────┐  push (collector)   ┌───────────────────────────────┐
 │ apps, deploy-│ ───────────────────▶│ deploygrid server             │
 │ ments,       │                     │  • reads System / Component / │
 │ ingresses    │ ◀─ pull (kubeconfig)│    Cluster custom resources   │
 └──────────────┘                     │  • builds the grid            │
                                      │  • writes Component.status    │
                                      │  • serves /api and the UI     │
                                      └───────────────────────────────┘
```

* **System** – one grid: ordered environments (columns) and groups (row sections).
* **Component** – one row, with a stable identity across environments. Declared
  as a resource, or *discovered* from labels on what is running; discovered
  rows are materialised as Component resources (`status.discovered: true`) so
  they can be enriched in place (`spec.discovery.createComponents` on the
  System turns this off).
* **Cluster** – a registered cluster: its API addresses (so Argo CD
  destinations resolve), the environment its namespaces map to, and how it is
  observed.
* **Configuration / ConfigurationView** – named values per system or
  environment, deep-merged, and templates that render them together with the
  grid (for example a hosts file or a values file per environment).

The full design, conventions and roadmap are in [V2DESIGN.md](V2DESIGN.md).

## Observing clusters

Two modes, usable together:

| Mode | Where the watch runs | Configure |
|---|---|---|
| **agent** (recommended) | A collector inside the observed cluster pushes to the server | `Cluster` resource with `collection.mode: agent`, then install `charts/deploygrid-collector` there |
| **kubeconfig** / **local** | The server watches the cluster itself | `clusters:` list in the server config (`local` uses the in-cluster service account) |

The images are distroless (API) and unprivileged nginx (UI, which proxies
`/api` to the API container in the same pod). There is no shell or cloud CLI
in the API image, so kubeconfigs that need a credential helper such as the
EKS `aws` exec plugin cannot be used in **kubeconfig** mode; use a collector
for those clusters.

For an agent-mode `Cluster` the server generates a token Secret named
`<cluster>-collector-token` in its namespace. Pass that token to the collector
chart:

```sh
TOKEN=$(kubectl -n deploygrid get secret prod-east-collector-token -o jsonpath='{.data.token}' | base64 -d)
helm install deploygrid-collector charts/deploygrid-collector \
  --set server.url=https://deploygrid.example.com/api \
  --set cluster.name=prod-east --set token=$TOKEN
```

## Operator-installed applications

Applications installed by an operator from a custom resource (for example a
`RiteSuite`) are observed by declaring the kind in `sources.applicationKinds`
on the server and on each collector:

```yaml
sources:
  applicationKinds:
    - group: platform.ritesuite.com
      version: v1alpha1
      resource: ritesuites
      kind: RiteSuite                                 # as in ownerReferences
      component: ritesuite                            # the row every instance maps to
      desiredVersionPath: "{.spec.version}"           # default
      runningVersionPath: "{.status.components[*].image}"  # optional
      pinnedVersionsPath: "{.spec.services.*.image.tag}"  # optional
      healthConditionType: Ready                      # default
```

How it works, and therefore what an operator must do for this to apply:

* **Identity.** One row per kind (`component`), or per
  `app.kubernetes.io/name` label / resource name when `component` is empty.
  Labels on the custom resource (`deploygrid.activated.io/component`,
  `/environment`) and declared `Component` selectors take precedence.
* **Environment.** From the resource's labels, `environmentPath`, the
  Cluster's namespace rules, or the cluster default, in that order. One
  instance per namespace per environment is the usual layout.
* **Workloads.** Deployments and Ingresses that carry a *controller
  ownerReference* to the custom resource are grouped under it (that is what
  `controller-runtime`'s `SetControllerReference` produces). Operators that
  create workloads without owner references are not linked; label the
  workloads with the deploygrid component instead.
* **Version.** Desired from `desiredVersionPath`. Running from
  `runningVersionPath` when the operator reports it (a version, or image
  references whose tags must agree), else the version most owned workloads
  actually run, else their `app.kubernetes.io/version` label (which usually
  records the intent, not what is serving, so it is the last resort).
  Workloads running different versions mark the cell inconsistent, except
  versions listed by `pinnedVersionsPath`, which an operator sets on
  purpose.
* **Health.** `Degraded=True` beats everything, then `<healthConditionType>=True`
  is healthy, `Progressing=True` is progressing, `<healthConditionType>=False`
  is degraded, no conditions is unknown.

The RBAC of both charts grows a read rule per declared kind.

## Labelling what you run

Identity is read from labels (selectable), display data from annotations:

```yaml
labels:
  deploygrid.activated.io/component: api          # row
  deploygrid.activated.io/environment: qa         # column (or let Cluster namespace rules decide)
  deploygrid.activated.io/system: shop            # only needed with several systems
annotations:
  deploygrid.activated.io/group: Backend
```

Workloads managed by Helm or Argo CD inherit the identity of their release or
Application, so labelling the Argo CD `Application` is usually enough. A
`Component` can also claim resources with `spec.selector`. Legacy
`deploygrid/name|environment|group` annotations still work and raise a
migration warning on the grid.

## API

```
GET  /api/systems                                   GET /api/systems/{s}
GET  /api/systems/{s}/grid                          GET /api/systems/{s}/components
GET  /api/systems/{s}/components/{c}                GET /api/systems/{s}/unassigned
GET  /api/systems/{s}/history                       GET /api/systems/{s}/components/{c}/history
GET  /api/systems/{s}/configurations                GET /api/systems/{s}/configurations/{name}?environment=
GET  /api/systems/{s}/views                         GET /api/systems/{s}/views/{view}?environment=
POST /api/observations        (collectors, bearer token)
GET  /swagger.json

Version changes are kept in memory for the API and recorded as Kubernetes
Events on the Component (`kubectl describe component <name>`).
```

## Development

```sh
make dev_kind         # three kind clusters: ops (control + Argo CD apps), two app clusters
make serve            # API on 127.0.0.1:8080 (CORS open for the Vite dev server), control cluster = kind-ops-cluster-1
make dev_collector    # push kind-app-cluster-2 into the running server
cd ui && npm run dev  # UI on http://localhost:5173 against the API
make test lint        # integration tests need the kind clusters
```

Configuration is YAML (`--config` or `$CONFIG_PATH`) with every key
overridable as `DEPLOYGRID_<SECTION>_<KEY>`, for example
`DEPLOYGRID_SERVER_PORT`. The `collector` subcommand reads the `collector`
section (`DEPLOYGRID_COLLECTOR_SERVER`, `_CLUSTER`, `_TOKEN` or `_TOKEN_FILE`).
