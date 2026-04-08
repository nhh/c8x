# c8x

Deploy and manage type-safe Kubernetes apps with TypeScript.

## Quick Start

```bash
# Install the CLI
npm install -g @c8x/cli

# Create a project
mkdir my-infra && cd my-infra
npm init -y
npm install c8x @c8x/wordpress

# Create chart.ts
cat > chart.ts << 'EOF'
import { Chart } from "c8x";
import wordpress from "@c8x/wordpress";

export default (): Chart => wordpress();
EOF

# Configure
echo "C8X_WP_DOMAIN=blog.example.com" > .env
echo "C8X_WP_DB_PASSWORD=secure-password" >> .env

# Deploy
c8x install chart.ts
```

## Commands

```
c8x install <file>          Deploy a chart and create release v1
c8x upgrade <file>          Upgrade an existing release
c8x rollback <name> [rev]   Rollback to a previous revision
c8x uninstall <name>        Delete all resources and release history
c8x inspect <file>          Preview generated YAML in the terminal
c8x diff <file>             Show what would change on upgrade
c8x history <name>          Show release history
c8x status <name>           Show current release status
c8x init <name>             Scaffold a new chart
c8x version                 Print version info
```

## Permissions

Charts run in a sandboxed environment. Access to external resources requires explicit flags (like Deno):

```
c8x install chart.ts                          # safe globals only
c8x install chart.ts --allow-file             # allow $file.read
c8x install chart.ts --allow-http             # allow $http requests
c8x install chart.ts --allow-cluster          # allow $cluster queries
c8x install chart.ts --allow-all              # allow everything
```

## Chart Globals

### Always available

| Global | Methods | Description |
|---|---|---|
| `$env` | `get(name)`, `getAsObject(prefix)` | Environment variables (C8X_ prefix) |
| `$chart` | `.name`, `.version`, `.appVersion` | Chart metadata from package.json |
| `$base64` | `encode(str)`, `decode(str)` | Base64 encode/decode |
| `$hash` | `sha256(str)`, `md5(str)` | Deterministic hashes |
| `$log` | `info(msg)`, `warn(msg)`, `error(msg)` | Output during compilation |
| `$assert` | `$assert(condition, msg)` | Fail fast if condition is falsy |
| `$yaml` | `parse(str)`, `stringify(obj)` | YAML <-> JS object |

### Requires `--allow-file`

| Global | Methods | Description |
|---|---|---|
| `$file` | `read(path)`, `exists(path)` | Read files from chart directory (path traversal protected) |

### Requires `--allow-http`

| Global | Methods | Description |
|---|---|---|
| `$http` | `get`, `getText`, `getJSON`, `post`, `postJSON` | HTTP requests at compile time |

### Requires `--allow-cluster`

| Global | Methods | Description |
|---|---|---|
| `$cluster` | `version()`, `versionAtLeast(v)`, `nodeCount()`, `apiAvailable(api)`, `crdExists(name)`, `exists(api, kind, ns, name)`, `list(kind, ns?)` | Query Kubernetes API |
| `$release` | `.exists`, `.revision`, `.status`, `.chartName`, `.chartVersion`, `.env`, `.deployedAt` | Current release state |

## Examples

### Validation before deploy

```typescript
const password = $env.get<string>("DB_PASSWORD") ?? "changeme";
$assert(password !== "changeme", "Set C8X_DB_PASSWORD to a real value");
$assert(password.length >= 12, "DB_PASSWORD must be at least 12 chars");
```

### TLS certificates from files

```typescript
// c8x install chart.ts --allow-file
const cert = $file.read("certs/tls.crt");
const key = $file.read("certs/tls.key");

components: [{
  apiVersion: "v1", kind: "Secret", type: "kubernetes.io/tls",
  metadata: { name: "app-tls" },
  data: { "tls.crt": $base64.encode(cert), "tls.key": $base64.encode(key) }
}]
```

### Config hash for rolling updates

```typescript
const configHash = $hash.sha256(JSON.stringify(configData));

metadata: {
  annotations: { "c8x/config-hash": configHash }
}
```

### External secrets from Vault

```typescript
// c8x install chart.ts --allow-http
const secret = $http.getJSON("https://vault.internal/v1/secret/data/db", {
  headers: { "X-Vault-Token": $env.get("VAULT_TOKEN") }
});
```

### Cluster-adaptive charts

```typescript
// c8x install chart.ts --allow-cluster
const useGateway = $cluster.apiAvailable("gateway.networking.k8s.io/v1");
const ingress = useGateway
  ? { apiVersion: "gateway.networking.k8s.io/v1", kind: "HTTPRoute", ... }
  : { apiVersion: "networking.k8s.io/v1", kind: "Ingress", ... };
```

### Conditional migration on first install

```typescript
// c8x install chart.ts --allow-cluster
if (!$release.exists) {
  components.push(MigrationJob());
}
```

### Read and modify YAML configs

```typescript
// c8x install chart.ts --allow-file
const config = $yaml.parse($file.read("prometheus.yml"));
config.scrape_configs = config.scrape_configs.concat([
  { job_name: "my-app", static_configs: [{ targets: ["app:8080"] }] }
]);

data: { "prometheus.yml": $yaml.stringify(config) }
```

## Available Charts

| Chart | Description |
|---|---|
| [@c8x/whoami](npm/charts/whoami/) | Minimal example (traefik/whoami) |
| [@c8x/wordpress](npm/charts/wordpress/) | WordPress with MariaDB |
| [@c8x/nextcloud](npm/charts/nextcloud/) | Nextcloud with PostgreSQL |
| [@c8x/youtrack](npm/charts/youtrack/) | JetBrains YouTrack |
| [@c8x/monitoring](npm/charts/monitoring/) | Prometheus + Grafana + node-exporter |

## Release Lifecycle

c8x stores release state as ConfigMaps in the target namespace:

```
kubectl get configmaps -l app.kubernetes.io/managed-by=c8x

NAME                          DATA   AGE
c8x.release.wordpress.v1     1      3d    # superseded
c8x.release.wordpress.v2     1      1d    # superseded
c8x.release.wordpress.v3     1      2h    # deployed
```

Each revision stores the rendered manifest, environment values, chart info, and timestamp. This enables `upgrade`, `rollback`, `history`, `status`, and `diff`.

## Why TypeScript?

| Topic | Helm | c8x |
|---|---|---|
| Packaging | Custom | npm |
| Templating | Go templates | TypeScript |
| Configuration | `--set servers.foo.port=80` | .env |
| Type safety | None | Full IDE support |
| Code sharing | Copy _helpers.tpl | npm packages |
| Validation | At apply time | At compile time (`$assert`) |
| Cluster awareness | None | `$cluster` API |

## Goals

Reuse existing infrastructure (npm, TypeScript, .env) for an enhanced developer experience deploying to Kubernetes.

## Non Goals

- Replace Helm for every use case

## Prior Art

- [Mill](https://mill-build.org/mill/depth/why-scala.html) – Scala as configuration
- [Pulumi](https://www.pulumi.com/) – Infrastructure as code in real languages
- [Spring JavaConfig](https://docs.spring.io/spring-javaconfig/docs/1.0.0.M4/reference/htmlsingle/spring-javaconfig-reference.html) – Code over XML since 2008
- [Yoke](https://github.com/yokecd/yoke) – Similar concept for K8s
