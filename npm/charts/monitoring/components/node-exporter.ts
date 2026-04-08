import { DaemonSet } from "c8x";

export default (): DaemonSet => ({
  apiVersion: "apps/v1",
  kind: "DaemonSet",
  metadata: { name: "node-exporter" },
  spec: {
    selector: { matchLabels: { app: "node-exporter" } },
    template: {
      metadata: {
        labels: { app: "node-exporter" },
        annotations: { "prometheus.io/scrape": "true", "prometheus.io/port": "9100" },
      },
      spec: {
        hostNetwork: true,
        hostPID: true,
        containers: [
          {
            name: "node-exporter",
            image: "prom/node-exporter:v1.8.2",
            args: [
              "--path.procfs=/host/proc",
              "--path.sysfs=/host/sys",
              "--path.rootfs=/host/root",
              "--collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+|var/lib/kubelet/.+)($|/)",
            ],
            ports: [{ containerPort: 9100, protocol: "TCP" }],
            volumeMounts: [
              { name: "proc", mountPath: "/host/proc", readOnly: true },
              { name: "sys", mountPath: "/host/sys", readOnly: true },
              { name: "root", mountPath: "/host/root", readOnly: true },
            ],
          },
        ],
        volumes: [
          { name: "proc", hostPath: { path: "/proc" } },
          { name: "sys", hostPath: { path: "/sys" } },
          { name: "root", hostPath: { path: "/" } },
        ],
      },
    },
  },
});
