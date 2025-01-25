import {Service} from "c8x";

export default (): Service => ({
  apiVersion: "v1",
  kind: "Service",
  spec: {
    selector: { app: "whoami" },
    ports: [{ port: 80 }],
    type: "ClusterIP",
  },
  metadata: {
    name: "whoami-svc",
  },
});
