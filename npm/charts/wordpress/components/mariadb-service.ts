import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: { name: "wordpress-db" },
  spec: {
    selector: { app: "wordpress-db" },
    ports: [{ port: 3306, targetPort: 3306 }],
    type: "ClusterIP",
  },
});
