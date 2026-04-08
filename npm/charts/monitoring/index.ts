import { Chart } from "c8x";

import { serviceAccount, clusterRole, clusterRoleBinding } from "./components/prometheus-rbac";
import PrometheusConfig from "./components/prometheus-config";
import Prometheus from "./components/prometheus";
import PrometheusService from "./components/prometheus-service";
import NodeExporter from "./components/node-exporter";
import NodeExporterService from "./components/node-exporter-service";
import GrafanaSecret from "./components/grafana-secret";
import GrafanaDatasources from "./components/grafana-config";
import GrafanaPvc from "./components/grafana-pvc";
import Grafana from "./components/grafana";
import GrafanaService from "./components/grafana-service";
import GrafanaIngress from "./components/grafana-ingress";

const values = {
  namespace: $env.get<string>("MON_NAMESPACE") ?? "monitoring",
  grafana: {
    domain: $env.get<string>("MON_GRAFANA_DOMAIN") ?? "grafana.example.com",
    adminPassword: $env.get<string>("MON_GRAFANA_ADMIN_PASSWORD") ?? "admin",
    storageSize: $env.get<string>("MON_GRAFANA_STORAGE_SIZE") ?? "5Gi",
  },
  prometheus: {
    storageSize: $env.get<string>("MON_PROMETHEUS_STORAGE_SIZE") ?? "20Gi",
    retention: $env.get<string>("MON_PROMETHEUS_RETENTION") ?? "15d",
  },
  storageClass: $env.get<string>("MON_STORAGE_CLASS") ?? "default",
};

export default (): Chart => ({
  namespace: {
    apiVersion: "v1",
    kind: "Namespace",
    metadata: { name: values.namespace },
  },
  components: [
    // Prometheus RBAC
    serviceAccount(),
    clusterRole(),
    clusterRoleBinding(),

    // Prometheus
    PrometheusConfig(),
    Prometheus({
      storageSize: values.prometheus.storageSize,
      storageClass: values.storageClass,
      retention: values.prometheus.retention,
    }),
    PrometheusService(),

    // Node Exporter (DaemonSet on every node)
    NodeExporter(),
    NodeExporterService(),

    // Grafana
    GrafanaSecret({ adminPassword: values.grafana.adminPassword }),
    GrafanaDatasources(),
    GrafanaPvc({
      storageSize: values.grafana.storageSize,
      storageClass: values.storageClass,
    }),
    Grafana({
      storageSize: values.grafana.storageSize,
      storageClass: values.storageClass,
    }),
    GrafanaService(),
    GrafanaIngress({ domain: values.grafana.domain }),
  ],
});
