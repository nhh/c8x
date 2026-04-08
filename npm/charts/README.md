# c8x Charts

Type-safe Kubernetes charts written in TypeScript, deployed with [c8x](https://github.com/nhh/c8x).

## Getting started

```bash
# Install the CLI
npm install -g @c8x/cli

# Create a project
mkdir my-infra && cd my-infra
npm init -y

# Install a chart
npm install c8x @c8x/wordpress
```

Create a `chart.ts`:

```typescript
import { Chart } from "c8x";
import wordpress from "@c8x/wordpress";

export default (): Chart => wordpress();
```

Create a `.env` with your configuration, then deploy:

```bash
c8x inspect chart.ts   # preview generated YAML
c8x install chart.ts   # deploy to your cluster
```

## Charts

### Web Applications

| Chart | Description | Version |
|---|---|---|
| [@c8x/wordpress](wordpress/) | WordPress with MariaDB, PVC, and Ingress | 0.0.1 |
| [@c8x/nextcloud](nextcloud/) | Nextcloud with PostgreSQL, PVC, and Ingress | 0.0.1 |

### Developer Tools

| Chart | Description | Version |
|---|---|---|
| [@c8x/youtrack](youtrack/) | JetBrains YouTrack issue tracker | 0.0.1 |

### Observability

| Chart | Description | Version |
|---|---|---|
| [@c8x/monitoring](monitoring/) | Prometheus + Grafana + node-exporter | 0.0.1 |

### Examples

| Chart | Description | Version |
|---|---|---|
| [@c8x/whoami](whoami/) | Minimal example chart (traefik/whoami) | 0.0.7 |

## Writing your own chart

A c8x chart is an npm package that exports a function returning a `Chart` object. Each component is a typed Kubernetes resource.

```bash
c8x init my-chart
cd my-chart
npm install
```

```
my-chart/
├── package.json
├── chart.ts
└── components/
    ├── deployment.ts
    ├── service.ts
    └── ingress.ts
```

```typescript
// components/deployment.ts
import { Deployment } from "c8x";

export default (): Deployment => ({
  apiVersion: "apps/v1",
  kind: "Deployment",
  metadata: { name: "my-app" },
  spec: {
    replicas: 1,
    selector: { matchLabels: { app: "my-app" } },
    template: {
      metadata: { labels: { app: "my-app" } },
      spec: {
        containers: [{ name: "app", image: "nginx" }],
      },
    },
  },
});
```

Publish it:

```bash
npm publish --access public
```

## Supported Kubernetes types

All components get full IDE autocompletion. Supported types:

| Group | Types |
|---|---|
| core/v1 | Namespace, Service, ConfigMap, Secret, PersistentVolumeClaim, PersistentVolume, ServiceAccount, Pod, Endpoints, LimitRange, ResourceQuota |
| apps/v1 | Deployment, StatefulSet, DaemonSet, ReplicaSet |
| batch/v1 | Job, CronJob |
| networking/v1 | Ingress, IngressClass, NetworkPolicy |
| rbac/v1 | Role, ClusterRole, RoleBinding, ClusterRoleBinding |
| autoscaling/v2 | HorizontalPodAutoscaler |
| policy/v1 | PodDisruptionBudget |
| storage/v1 | StorageClass |
