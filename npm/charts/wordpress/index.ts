import { Chart } from "c8x";

import DbSecret from "./components/secret";
import MariaDb from "./components/mariadb";
import MariaDbService from "./components/mariadb-service";
import Pvc from "./components/pvc";
import WordpressDeployment from "./components/deployment";
import WordpressService from "./components/service";
import WordpressIngress from "./components/ingress";

const values = {
  namespace: $env.get<string>("WP_NAMESPACE") ?? "wordpress",
  domain: $env.get<string>("WP_DOMAIN") ?? "blog.example.com",
  replicas: $env.get<number>("WP_REPLICAS") ?? 1,
  image: $env.get<string>("WP_IMAGE") ?? "wordpress:6.7-apache",
  storageSize: $env.get<string>("WP_STORAGE_SIZE") ?? "10Gi",
  storageClass: $env.get<string>("WP_STORAGE_CLASS") ?? "default",
  db: {
    name: $env.get<string>("WP_DB_NAME") ?? "wordpress",
    user: $env.get<string>("WP_DB_USER") ?? "wordpress",
    password: $env.get<string>("WP_DB_PASSWORD") ?? "changeme",
    rootPassword: $env.get<string>("WP_DB_ROOT_PASSWORD") ?? "rootchangeme",
    storageSize: $env.get<string>("WP_DB_STORAGE_SIZE") ?? "5Gi",
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
      dbRootPassword: values.db.rootPassword,
    }),
    Pvc({ storageSize: values.storageSize, storageClass: values.storageClass }),
    MariaDb({ storageSize: values.db.storageSize, storageClass: values.storageClass }),
    MariaDbService(),
    WordpressDeployment({
      replicas: values.replicas,
      image: values.image,
      domain: values.domain,
    }),
    WordpressService(),
    WordpressIngress({ domain: values.domain }),
  ],
});
