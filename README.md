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
  as a resource, or *discovered* from labels on what is running.
* **Cluster** – a registered cluster: its API addresses (so Argo CD
  destinations resolve), the environment its namespaces map to, and how it is
  observed.
* **Configuration / ConfigurationView** – named values per system or
  environment and templates that render them (phase 3).

The full design, conventions and roadmap are in [V2DESIGN.md](V2DESIGN.md).

## Observing clusters

Two modes, usable together:

| Mode | Where the watch runs | Configure |
|---|---|---|
| **agent** (recommended) | A collector inside the observed cluster pushes to the server | `Cluster` resource with `collection.mode: agent`, then install `charts/deploygrid-collector` there |
| **kubeconfig** / **local** | The server watches the cluster itself | `clusters:` list in the server config (`local` uses the in-cluster service account) |

For an agent-mode `Cluster` the server generates a token Secret named
`<cluster>-collector-token` in its namespace. Pass that token to the collector
chart:

```sh
TOKEN=$(kubectl -n deploygrid get secret prod-east-collector-token -o jsonpath='{.data.token}' | base64 -d)
helm install deploygrid-collector charts/deploygrid-collector \
  --set server.url=https://deploygrid.example.com/api \
  --set cluster.name=prod-east --set token=$TOKEN
```

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
POST /api/observations        (collectors, bearer token)
GET  /swagger.json
```

## Development

```sh
make dev_kind         # three kind clusters: ops (control + Argo CD apps), two app clusters
make serve            # API on 127.0.0.1:8080, control cluster = kind-ops-cluster-1
make dev_collector    # push kind-app-cluster-2 into the running server
cd ui && npm run dev  # UI on http://localhost:5173 against the API
make test lint        # integration tests need the kind clusters
```

Configuration is YAML (`--config` or `$CONFIG_PATH`) with every key
overridable as `DEPLOYGRID_<SECTION>_<KEY>`, for example
`DEPLOYGRID_SERVER_PORT`. The `collector` subcommand reads the `collector`
section (`DEPLOYGRID_COLLECTOR_SERVER`, `_CLUSTER`, `_TOKEN` or `_TOKEN_FILE`).
