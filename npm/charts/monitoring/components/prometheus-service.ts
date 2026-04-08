import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: { name: "prometheus" },
  spec: {
    selector: { app: "prometheus" },
    ports: [{ port: 9090, targetPort: 9090 }],
    type: "ClusterIP",
  },
});
