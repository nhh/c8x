import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: { name: "grafana" },
  spec: {
    selector: { app: "grafana" },
    ports: [{ port: 3000, targetPort: 3000 }],
    type: "ClusterIP",
  },
});
