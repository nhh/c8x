import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: { name: "youtrack" },
  spec: {
    selector: { app: "youtrack" },
    ports: [{ port: 8080, targetPort: 8080 }],
    type: "ClusterIP",
  },
});
