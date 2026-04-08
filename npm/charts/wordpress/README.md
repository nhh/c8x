# @c8x/wordpress

WordPress deployment for Kubernetes, powered by [c8x](https://github.com/nhh/c8x).

Deploys WordPress with a dedicated MariaDB database, persistent storage, and NGINX Ingress with TLS.

## Installation

```bash
npm install @c8x/wordpress
```

## Usage

Create a `chart.ts` in your project:

```typescript
import { Chart } from "c8x";
import wordpress from "@c8x/wordpress";

export default (): Chart => wordpress();
```

Create a `.env` file next to it:

```bash
C8X_WP_NAMESPACE=wordpress
C8X_WP_DOMAIN=blog.example.com
C8X_WP_DB_PASSWORD=a-secure-password
C8X_WP_DB_ROOT_PASSWORD=a-secure-root-password
```

Deploy:

```bash
c8x inspect chart.ts   # preview
c8x install chart.ts   # deploy
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `C8X_WP_NAMESPACE` | `wordpress` | Kubernetes namespace |
| `C8X_WP_DOMAIN` | `blog.example.com` | Domain for Ingress and WP_HOME/WP_SITEURL |
| `C8X_WP_REPLICAS` | `1` | Number of WordPress replicas |
| `C8X_WP_IMAGE` | `wordpress:6.7-apache` | WordPress container image |
| `C8X_WP_STORAGE_SIZE` | `10Gi` | Storage for WordPress data |
| `C8X_WP_STORAGE_CLASS` | `default` | StorageClass for all PVCs |
| `C8X_WP_DB_NAME` | `wordpress` | MariaDB database name |
| `C8X_WP_DB_USER` | `wordpress` | MariaDB user |
| `C8X_WP_DB_PASSWORD` | `changeme` | MariaDB password |
| `C8X_WP_DB_ROOT_PASSWORD` | `rootchangeme` | MariaDB root password |
| `C8X_WP_DB_STORAGE_SIZE` | `5Gi` | Storage for MariaDB data |

## What gets deployed

| Kind | Name | Description |
|---|---|---|
| Secret | wordpress-db-credentials | MariaDB credentials |
| PersistentVolumeClaim | wordpress-data | WordPress `/var/www/html` |
| StatefulSet | wordpress-db | MariaDB 11 with volume claim template |
| Service | wordpress-db | ClusterIP for MariaDB on port 3306 |
| Deployment | wordpress | WordPress container with secretKeyRef for DB credentials |
| Service | wordpress | ClusterIP on port 80 |
| Ingress | wordpress | NGINX Ingress with TLS and 64m upload limit |
