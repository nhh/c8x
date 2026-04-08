import { StatefulSet } from "c8x";

export type PrometheusProps = {
  storageSize: string;
  storageClass: string;
  retention: string;
};

export default (props: PrometheusProps): StatefulSet => ({
  apiVersion: "apps/v1",
  kind: "StatefulSet",
  metadata: { name: "prometheus" },
  spec: {
    serviceName: "prometheus",
    replicas: 1,
    selector: { matchLabels: { app: "prometheus" } },
    template: {
      metadata: { labels: { app: "prometheus" } },
      spec: {
        serviceAccountName: "prometheus",
        securityContext: { fsGroup: 65534, runAsNonRoot: true, runAsUser: 65534 },
        containers: [
          {
            name: "prometheus",
            image: "prom/prometheus:v2.55.0",
            args: [
              "--config.file=/etc/prometheus/prometheus.yml",
              "--storage.tsdb.path=/prometheus",
              `--storage.tsdb.retention.time=${props.retention}`,
              "--web.enable-lifecycle",
            ],
            ports: [{ containerPort: 9090, protocol: "TCP" }],
            volumeMounts: [
              { name: "prometheus-config", mountPath: "/etc/prometheus" },
              { name: "prometheus-data", mountPath: "/prometheus" },
            ],
          },
        ],
        volumes: [
          {
            name: "prometheus-config",
            configMap: { name: "prometheus-config" },
          },
        ],
      },
    },
    volumeClaimTemplates: [
      {
        metadata: { name: "prometheus-data" },
        spec: {
          accessModes: ["ReadWriteOnce"],
          storageClassName: props.storageClass,
          resources: { requests: { storage: props.storageSize } },
        },
      },
    ],
  },
});
