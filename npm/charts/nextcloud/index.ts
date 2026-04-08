import { Chart } from "c8x";

import NextcloudDeployment from "./components/deployment";
import NextcloudService from "./components/service";
import NextcloudIngress from "./components/ingress";
import Postgres from "./components/postgres";
import PostgresService from "./components/postgres-service";
import Pvc from "./components/pvc";
import DbSecret from "./components/secret";
import NextcloudConfig from "./components/configmap";

const values = {
  namespace: $env.get<string>("NC_NAMESPACE") ?? "nextcloud",
  domain: $env.get<string>("NC_DOMAIN") ?? "cloud.example.com",
  replicas: $env.get<number>("NC_REPLICAS") ?? 1,
  image: $env.get<string>("NC_IMAGE") ?? "nextcloud:29-apache",
  storageSize: $env.get<string>("NC_STORAGE_SIZE") ?? "10Gi",
  storageClass: $env.get<string>("NC_STORAGE_CLASS") ?? "default",
  db: {
    name: $env.get<string>("NC_DB_NAME") ?? "nextcloud",
    user: $env.get<string>("NC_DB_USER") ?? "nextcloud",
    password: $env.get<string>("NC_DB_PASSWORD") ?? "changeme",
    storageSize: $env.get<string>("NC_DB_STORAGE_SIZE") ?? "5Gi",
  },
};

export default (): Chart => ({
  namespace: {
    apiVersion: "v1",
    kind: "Namespace",
    metadata: { name: values.namespace },
  },
  components: [
    DbSecret({
      dbName: values.db.name,
      dbUser: values.db.user,
      dbPassword: values.db.password,
    }),
    NextcloudConfig({
      domain: values.domain,
      dbName: values.db.name,
      dbUser: values.db.user,
    }),
    Pvc({
      storageSize: values.storageSize,
      storageClass: values.storageClass,
    }),
    Postgres({
      storageSize: values.db.storageSize,
      storageClass: values.storageClass,
    }),
    PostgresService(),
    NextcloudDeployment({
      replicas: values.replicas,
      image: values.image,
    }),
    NextcloudService(),
    NextcloudIngress({ domain: values.domain }),
  ],
});
