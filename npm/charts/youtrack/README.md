# @c8x/youtrack

JetBrains YouTrack deployment for Kubernetes, powered by [c8x](https://github.com/nhh/c8x).

Deploys YouTrack as a StatefulSet with persistent storage for data, config, logs, and backups.

## Installation

```bash
npm install @c8x/youtrack
```

## Usage

Create a `chart.ts` in your project:

```typescript
import { Chart } from "c8x";
import youtrack from "@c8x/youtrack";

export default (): Chart => youtrack();
```

Create a `.env` file next to it:

```bash
C8X_YT_NAMESPACE=youtrack
C8X_YT_DOMAIN=youtrack.example.com
C8X_YT_BASE_URL=https://youtrack.example.com
```

Deploy:

```bash
c8x inspect chart.ts   # preview
c8x install chart.ts   # deploy
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `C8X_YT_NAMESPACE` | `youtrack` | Kubernetes namespace |
| `C8X_YT_DOMAIN` | `youtrack.example.com` | Domain for Ingress |
| `C8X_YT_IMAGE` | `jetbrains/youtrack:2024.3` | YouTrack container image |
| `C8X_YT_STORAGE_SIZE` | `10Gi` | Storage for YouTrack data |
| `C8X_YT_STORAGE_CLASS` | `default` | StorageClass for all PVCs |
| `C8X_YT_BASE_URL` | `https://youtrack.example.com` | YouTrack base URL |

## What gets deployed

| Kind | Name | Description |
|---|---|---|
| ConfigMap | youtrack-config | Base URL configuration |
| StatefulSet | youtrack | YouTrack with 4 volume claim templates (data, conf, logs, backups) |
| Service | youtrack | ClusterIP on port 8080 |
| Ingress | youtrack | NGINX Ingress with TLS, 100m upload, 3600s timeouts |
