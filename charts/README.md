# k8x charts

## How it looks?
[whoami chart](/whoami/chart.ts).
```ts
/// <reference types="@kubernetix/types" />

import MyIngress, { MyIngressProps } from "./components/ingress";
import Deployment, { MyDeploymentProps } from "./components/deployment";
import Service from "./components/service";

const values: {
  namespace: string;
  ingress: MyIngressProps;
  deployment: MyDeploymentProps;
} = {
  namespace: $env.get<string>("WHOAMI_NAMESPACE") ?? "default",
  deployment: {
    replicas: $env.get<number>("WHOAMI_REPLICAS") ?? 1,
  },
  ingress: {
    domain: $env.get<string>("WHOAMI_DOMAIN") ?? "example.com",
  },
};

export default (): k8x.Chart => ({
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

```