import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: { name: "wordpress" },
  spec: {
    selector: { app: "wordpress" },
    ports: [{ port: 80, targetPort: 80 }],
    type: "ClusterIP",
  },
});
