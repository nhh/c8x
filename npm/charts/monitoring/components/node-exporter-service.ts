import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: { name: "node-exporter" },
  spec: {
    selector: { app: "node-exporter" },
    ports: [{ port: 9100, targetPort: 9100 }],
    type: "ClusterIP",
  },
});
