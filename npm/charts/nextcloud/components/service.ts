import { Service } from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  metadata: {
    name: "nextcloud",
  },
  spec: {
    selector: { app: "nextcloud" },
    ports: [{ port: 80, targetPort: 80 }],
    type: "ClusterIP",
  },
});
