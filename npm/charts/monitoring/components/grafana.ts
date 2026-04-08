import { Deployment } from "c8x";

export type GrafanaProps = {
  storageSize: string;
  storageClass: string;
};

export default (props: GrafanaProps): Deployment => ({
  apiVersion: "apps/v1",
  kind: "Deployment",
  metadata: { name: "grafana" },
  spec: {
    replicas: 1,
    selector: { matchLabels: { app: "grafana" } },
    template: {
      metadata: { labels: { app: "grafana" } },
      spec: {
        securityContext: { fsGroup: 472, runAsUser: 472 },
        containers: [
          {
            name: "grafana",
            image: "grafana/grafana:11.3.0",
            ports: [{ containerPort: 3000, protocol: "TCP" }],
            envFrom: [{ secretRef: { name: "grafana-credentials" } }],
            volumeMounts: [
              { name: "grafana-data", mountPath: "/var/lib/grafana" },
              { name: "grafana-datasources", mountPath: "/etc/grafana/provisioning/datasources" },
            ],
          },
        ],
        volumes: [
          {
            name: "grafana-data",
            persistentVolumeClaim: { claimName: "grafana-data" },
          },
          {
            name: "grafana-datasources",
            configMap: { name: "grafana-datasources" },
          },
        ],
      },
    },
  },
});
