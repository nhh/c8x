import { StatefulSet } from "c8x";

export type YouTrackProps = {
  image: string;
  storageSize: string;
  storageClass: string;
};

export default (props: YouTrackProps): StatefulSet => ({
  apiVersion: "apps/v1",
  kind: "StatefulSet",
  metadata: { name: "youtrack" },
  spec: {
    serviceName: "youtrack",
    replicas: 1,
    selector: { matchLabels: { app: "youtrack" } },
    template: {
      metadata: { labels: { app: "youtrack" } },
      spec: {
        containers: [
          {
            name: "youtrack",
            image: props.image,
            ports: [{ containerPort: 8080, protocol: "TCP" }],
            envFrom: [{ configMapRef: { name: "youtrack-config" } }],
            volumeMounts: [
              { name: "youtrack-data", mountPath: "/opt/youtrack/data" },
              { name: "youtrack-conf", mountPath: "/opt/youtrack/conf" },
              { name: "youtrack-logs", mountPath: "/opt/youtrack/logs" },
              { name: "youtrack-backups", mountPath: "/opt/youtrack/backups" },
            ],
          },
        ],
        securityContext: {
          fsGroup: 13001,
        },
      },
    },
    volumeClaimTemplates: [
      {
        metadata: { name: "youtrack-data" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: { requests: { storage: props.storageSize } },
        },
      },
      {
        metadata: { name: "youtrack-conf" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: { requests: { storage: "1Gi" } },
        },
      },
      {
        metadata: { name: "youtrack-logs" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: { requests: { storage: "2Gi" } },
        },
      },
      {
        metadata: { name: "youtrack-backups" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: { requests: { storage: "10Gi" } },
        },
      },
    ],
  },
});
