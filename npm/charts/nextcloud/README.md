# @c8x/nextcloud

Nextcloud deployment for Kubernetes, powered by [c8x](https://github.com/nhh/c8x).

Deploys Nextcloud with a dedicated PostgreSQL database, persistent storage, and NGINX Ingress with TLS.

## Prerequisites

- Kubernetes cluster (>= 1.31)
- NGINX Ingress Controller
- A StorageClass that supports `ReadWriteOnce`
- [c8x](https://github.com/nhh/c8x) CLI installed

## Installation

```bash
npm install @c8x/nextcloud
```

## Usage

Create a `chart.ts` in your project:

```typescript
import { Chart } from "c8x";
import nextcloud from "@c8x/nextcloud";

export default (): Chart => nextcloud();
```

Create a `.env` file next to it:

```bash
C8X_NC_NAMESPACE=nextcloud
C8X_NC_DOMAIN=cloud.example.com
C8X_NC_DB_PASSWORD=a-secure-password
```

Preview the generated Kubernetes YAML:

```bash
c8x inspect chart.ts
```

Deploy to your cluster:

```bash
c8x install chart.ts
```

## Configuration

All values are configured via environment variables with the `C8X_` prefix. Place them in a `.env` file next to your `chart.ts`.

| Variable | Default | Description |
|---|---|---|
| `C8X_NC_NAMESPACE` | `nextcloud` | Kubernetes namespace |
| `C8X_NC_DOMAIN` | `cloud.example.com` | Domain for Ingress and trusted domains |
| `C8X_NC_REPLICAS` | `1` | Number of Nextcloud replicas |
| `C8X_NC_IMAGE` | `nextcloud:29-apache` | Nextcloud container image |
| `C8X_NC_STORAGE_SIZE` | `10Gi` | Storage size for Nextcloud data |
| `C8X_NC_STORAGE_CLASS` | `default` | StorageClass for all PVCs |
| `C8X_NC_DB_NAME` | `nextcloud` | PostgreSQL database name |
| `C8X_NC_DB_USER` | `nextcloud` | PostgreSQL user |
| `C8X_NC_DB_PASSWORD` | `changeme` | PostgreSQL password |
| `C8X_NC_DB_STORAGE_SIZE` | `5Gi` | Storage size for PostgreSQL data |

## Composing with other charts

Since `chart.ts` is your entry point, you can combine multiple charts or add your own components:

```typescript
import { Chart } from "c8x";
import nextcloud from "@c8x/nextcloud";

export default (): Chart => {
  const nc = nextcloud();

  return {
    ...nc,
    components: [
      ...nc.components,
      // Add your own resources here
      {
        apiVersion: "v1",
        kind: "ConfigMap",
        metadata: { name: "my-extra-config" },
        data: { CUSTOM_KEY: "value" },
      },
    ],
  };
};
```

## What gets deployed

| Component | Kind | Description |
|---|---|---|
| `secret.ts` | Secret | PostgreSQL credentials (user, password, database name) |
| `configmap.ts` | ConfigMap | Nextcloud configuration (trusted domains, DB host, overwrite protocol) |
| `pvc.ts` | PersistentVolumeClaim | Storage for Nextcloud data (`/var/www/html`) |
| `postgres.ts` | StatefulSet | PostgreSQL 16 with its own volume claim template |
| `postgres-service.ts` | Service | ClusterIP for PostgreSQL on port 5432 |
| `deployment.ts` | Deployment | Nextcloud container with envFrom for ConfigMap and Secret |
| `service.ts` | Service | ClusterIP for Nextcloud on port 80 |
| `ingress.ts` | Ingress | NGINX Ingress with TLS and 512m upload limit |

## Architecture

```
                    ┌─────────┐
                    │ Ingress │  cloud.example.com
                    └────┬────┘
                         │ :80
                  ┌──────┴──────┐
                  │  Service    │  nextcloud
                  └──────┬──────┘
                         │
                  ┌──────┴──────┐
                  │ Deployment  │  nextcloud:29-apache
                  │             │
                  │  envFrom:   │
                  │  - ConfigMap│
                  │  - Secret   │
                  └──────┬──────┘
                         │
              ┌──────────┼──────────┐
              │                     │
       ┌──────┴──────┐      ┌──────┴──────┐
       │     PVC     │      │   Service   │  nextcloud-db
       │  /var/www/  │      └──────┬──────┘
       │    html     │             │ :5432
       └─────────────┘      ┌──────┴──────┐
                            │ StatefulSet │  postgres:16-alpine
                            │             │
                            │  envFrom:   │
                            │  - Secret   │
                            └──────┬──────┘
                                   │
                            ┌──────┴──────┐
                            │     PVC     │
                            │  /var/lib/  │
                            │  postgresql │
                            └─────────────┘
```
