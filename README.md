# c8x

Kubernetes deployments in TypeScript. Type-safe, composable, from npm.

```typescript
// chart.ts
import { Chart } from "c8x";

export default (): Chart => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "app" } },
  components: [
    { apiVersion: "apps/v1", kind: "Deployment", metadata: { name: "web" },
      spec: { replicas: $env.get("REPLICAS") ?? 2,
        selector: { matchLabels: { app: "web" } },
        template: { metadata: { labels: { app: "web" } },
          spec: { containers: [{ name: "web", image: "nginx" }] } } } },
    { apiVersion: "v1", kind: "Service", metadata: { name: "web" },
      spec: { selector: { app: "web" }, ports: [{ port: 80 }] } },
  ],
});
```

```bash
c8x up chart.ts
```

That's it. TypeScript compiles, validates, and deploys to your cluster. Full IDE autocompletion for every Kubernetes field.

## Install

```bash
npm install -g @c8x/cli
```

## Usage

```bash
c8x up chart.ts              # deploy (install or upgrade, like docker compose up)
c8x down wordpress            # remove everything (like docker compose down)
c8x inspect chart.ts          # preview YAML without deploying
c8x diff chart.ts             # show what would change
c8x rollback wordpress        # rollback to previous revision
c8x history wordpress          # show all revisions
c8x status wordpress           # show current state
```

## Use charts from npm

```bash
npm install c8x @c8x/wordpress
```

```typescript
import { Chart } from "c8x";
import wordpress from "@c8x/wordpress";

export default (): Chart => wordpress();
```

```bash
echo "C8X_WP_DOMAIN=blog.example.com" > .env
c8x up chart.ts
```

## Available Charts

| Chart | Description |
|---|---|
| [@c8x/wordpress](npm/charts/wordpress/) | WordPress + MariaDB |
| [@c8x/nextcloud](npm/charts/nextcloud/) | Nextcloud + PostgreSQL |
| [@c8x/youtrack](npm/charts/youtrack/) | JetBrains YouTrack |
| [@c8x/monitoring](npm/charts/monitoring/) | Prometheus + Grafana + node-exporter |

## Why not Helm?

```yaml
# Helm: Go templates in YAML
{{- if and .Values.ingress.enabled (gt (len .Values.ingress.hosts) 0) }}
{{- range .Values.ingress.hosts }}
  - host: {{ .host | quote }}
    http:
      paths:
        {{- range .paths }}
        - path: {{ .path }}
        {{- end }}
{{- end }}
{{- end }}
```

```typescript
// c8x: just TypeScript
values.ingress.hosts
  .filter(h => h.enabled)
  .map(host => ({
    host: host.name,
    http: { paths: host.paths.map(p => ({ path: p.path, pathType: "Prefix" })) },
  }))
```

| | Helm | c8x |
|---|---|---|
| Templating | Go templates | TypeScript |
| Packaging | Custom | npm |
| Configuration | `--set x.y.z=1` | .env |
| Type safety | None | Full IDE support |
| Validation | At apply time | Before deploy (`$assert`) |
| Code sharing | Copy `_helpers.tpl` | `npm install` |
| Cluster awareness | None | `$cluster` API |

## Permissions

Charts run sandboxed. External access requires explicit flags (like Deno):

```bash
c8x up chart.ts                   # safe globals only
c8x up chart.ts --allow-file      # $file.read
c8x up chart.ts --allow-http      # $http requests
c8x up chart.ts --allow-cluster   # $cluster queries
c8x up chart.ts -A                # allow all
```

## Globals

**Always available:** `$env`, `$chart`, `$base64`, `$hash`, `$log`, `$assert`, `$yaml`

**With `--allow-file`:** `$file.read(path)`, `$file.exists(path)`

**With `--allow-http`:** `$http.get`, `$http.getJSON`, `$http.post`, `$http.postJSON`

**With `--allow-cluster`:** `$cluster.version()`, `$cluster.versionAtLeast(v)`, `$cluster.apiAvailable(api)`, `$cluster.nodeCount()`, `$cluster.exists(...)`, `$cluster.list(...)`, `$release.*`

## Examples

```typescript
// Validate before deploy
$assert($env.get("DB_PASSWORD") !== "changeme", "Set a real password");

// Load TLS certs from disk
data: { "tls.crt": $base64.encode($file.read("certs/tls.crt")) }

// Config hash for rolling updates
annotations: { "c8x/config-hash": $hash.sha256(JSON.stringify(config)) }

// Fetch secrets from Vault
const secret = $http.getJSON("https://vault.internal/v1/secret/data/db",
  { headers: { "X-Vault-Token": $env.get("VAULT_TOKEN") } });

// Adapt to cluster capabilities
const ingress = $cluster.apiAvailable("gateway.networking.k8s.io/v1")
  ? GatewayRoute({ domain })
  : Ingress({ domain });

// Migration only on first deploy
if (!$release.exists) components.push(MigrationJob());

// Conditional components
components: [
  App(values),
  ...(enableRedis ? [Redis()] : []),
]
```

## Release State

Stored as ConfigMaps in the namespace. Enables `up`, `down`, `rollback`, `history`, `status`, `diff`.

```bash
$ c8x history wordpress
REVISION   STATUS       TRIGGER    RESOURCES  DEPLOYER              DEPLOYED
1          superseded   ci         5          runner@gh-actions-12  2026-04-06 10:00:00
2          superseded   manual     5          niklas@mbp            2026-04-07 15:00:00
3          deployed     rollback   5          niklas@mbp            2026-04-08 22:00:00
```
