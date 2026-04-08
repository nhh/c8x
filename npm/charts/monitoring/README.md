# @c8x/monitoring

Prometheus + Grafana monitoring stack for Kubernetes, powered by [c8x](https://github.com/nhh/c8x).

Deploys Prometheus with node-exporter for metrics collection and Grafana with auto-provisioned Prometheus datasource.

## Installation

```bash
npm install @c8x/monitoring
```

## Usage

Create a `chart.ts` in your project:

```typescript
import { Chart } from "c8x";
import monitoring from "@c8x/monitoring";

export default (): Chart => monitoring();
```

Create a `.env` file next to it:

```bash
C8X_MON_NAMESPACE=monitoring
C8X_MON_GRAFANA_DOMAIN=grafana.example.com
C8X_MON_GRAFANA_ADMIN_PASSWORD=a-secure-password
```

Deploy:

```bash
c8x inspect chart.ts   # preview
c8x install chart.ts   # deploy
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `C8X_MON_NAMESPACE` | `monitoring` | Kubernetes namespace |
| `C8X_MON_GRAFANA_DOMAIN` | `grafana.example.com` | Domain for Grafana Ingress |
| `C8X_MON_GRAFANA_ADMIN_PASSWORD` | `admin` | Grafana admin password |
| `C8X_MON_GRAFANA_STORAGE_SIZE` | `5Gi` | Storage for Grafana dashboards/data |
| `C8X_MON_PROMETHEUS_STORAGE_SIZE` | `20Gi` | Storage for Prometheus TSDB |
| `C8X_MON_PROMETHEUS_RETENTION` | `15d` | Prometheus data retention period |
| `C8X_MON_STORAGE_CLASS` | `default` | StorageClass for all PVCs |

## What gets deployed

### Prometheus

| Kind | Name | Description |
|---|---|---|
| ServiceAccount | prometheus | Identity for Prometheus to scrape the K8s API |
| ClusterRole | prometheus | Permissions to read nodes, pods, services, endpoints, metrics |
| ClusterRoleBinding | prometheus | Binds ClusterRole to ServiceAccount |
| ConfigMap | prometheus-config | `prometheus.yml` with scrape configs for nodes, pods, and node-exporter |
| StatefulSet | prometheus | Prometheus v2.55.0 with persistent TSDB storage |
| Service | prometheus | ClusterIP on port 9090 |

### Node Exporter

| Kind | Name | Description |
|---|---|---|
| DaemonSet | node-exporter | Runs on every node, exposes host metrics on port 9100 |
| Service | node-exporter | ClusterIP for Prometheus to scrape |

### Grafana

| Kind | Name | Description |
|---|---|---|
| Secret | grafana-credentials | Admin user and password |
| ConfigMap | grafana-datasources | Auto-provisions Prometheus as default datasource |
| PersistentVolumeClaim | grafana-data | Storage for dashboards and settings |
| Deployment | grafana | Grafana v11.3.0 with provisioned datasource |
| Service | grafana | ClusterIP on port 3000 |
| Ingress | grafana | NGINX Ingress with TLS |

## Prometheus scrape targets

The default configuration scrapes:

- **prometheus** itself (`localhost:9090`)
- **node-exporter** endpoints (via kubernetes_sd_configs)
- **kubernetes-pods** with annotation `prometheus.io/scrape: "true"`
- **kubernetes-nodes** (kubelet metrics)

To scrape your application, add these annotations to your pod:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```
