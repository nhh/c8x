import {Chart} from "c8x"

import MyIngress from "./components/ingress";
import Deployment from "./components/deployment";
import Service from "./components/service";

const values = {
  namespace: $env.get<string>("WHOAMI_NAMESPACE") ?? "default",
  deployment: {
    replicas: $env.get<number>("WHOAMI_REPLICAS") ?? 1,
  },
  ingress: {
    domain: $env.get<string>("WHOAMI_DOMAIN") ?? "example.com",
  },
};

export default (): Chart => ({
  namespace: {
    kind: "Namespace",
    apiVersion: "v1",
    metadata: { name: values.namespace },
  },
  components: [
    MyIngress(values.ingress),
    Deployment(values.deployment),
    Service(),
  ],
});
