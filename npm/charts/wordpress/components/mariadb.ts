import { StatefulSet } from "c8x";

export type MariaDbProps = {
  storageSize: string;
  storageClass: string;
};

export default (props: MariaDbProps): StatefulSet => ({
  apiVersion: "apps/v1",
  kind: "StatefulSet",
  metadata: { name: "wordpress-db" },
  spec: {
    serviceName: "wordpress-db",
    replicas: 1,
    selector: { matchLabels: { app: "wordpress-db" } },
    template: {
      metadata: { labels: { app: "wordpress-db" } },
      spec: {
        containers: [
          {
            name: "mariadb",
            image: "mariadb:11",
            ports: [{ containerPort: 3306, protocol: "TCP" }],
            envFrom: [{ secretRef: { name: "wordpress-db-credentials" } }],
            volumeMounts: [
              { name: "mariadb-data", mountPath: "/var/lib/mysql" },
            ],
          },
        ],
      },
    },
    volumeClaimTemplates: [
      {
        metadata: { name: "mariadb-data" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: { requests: { storage: props.storageSize } },
        },
      },
    ],
  },
});
