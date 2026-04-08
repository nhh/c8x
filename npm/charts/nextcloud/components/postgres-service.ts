import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: {
    name: "nextcloud-db",
  },
  spec: {
    selector: { app: "nextcloud-db" },
    ports: [{ port: 5432, targetPort: 5432 }],
    type: "ClusterIP",
  },
});
