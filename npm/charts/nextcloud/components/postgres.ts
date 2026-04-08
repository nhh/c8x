import { StatefulSet } from "c8x";

export type PostgresProps = {
  storageSize: string;
  storageClass: string;
};

export default (props: PostgresProps): StatefulSet => ({
  apiVersion: "apps/v1",
  kind: "StatefulSet",
  metadata: {
    name: "nextcloud-db",
  },
  spec: {
    serviceName: "nextcloud-db",
    replicas: 1,
    selector: { matchLabels: { app: "nextcloud-db" } },
    template: {
      metadata: { labels: { app: "nextcloud-db" } },
      spec: {
        containers: [
          {
            name: "postgres",
            image: "postgres:16-alpine",
            ports: [{ containerPort: 5432, protocol: "TCP" }],
            envFrom: [
              { secretRef: { name: "nextcloud-db-credentials" } },
            ],
            volumeMounts: [
              { name: "postgres-data", mountPath: "/var/lib/postgresql/data" },
            ],
          },
        ],
      },
    },
    volumeClaimTemplates: [
      {
        metadata: { name: "postgres-data" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: {
            requests: {
              storage: props.storageSize,
            },
          },
        },
      },
    ],
  },
});
