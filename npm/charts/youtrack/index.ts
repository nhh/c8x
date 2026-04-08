import { Chart } from "c8x";

import YouTrackConfig from "./components/configmap";
import YouTrackStatefulSet from "./components/statefulset";
import YouTrackService from "./components/service";
import YouTrackIngress from "./components/ingress";

const values = {
  namespace: $env.get<string>("YT_NAMESPACE") ?? "youtrack",
  domain: $env.get<string>("YT_DOMAIN") ?? "youtrack.example.com",
  image: $env.get<string>("YT_IMAGE") ?? "jetbrains/youtrack:2024.3",
  storageSize: $env.get<string>("YT_STORAGE_SIZE") ?? "10Gi",
  storageClass: $env.get<string>("YT_STORAGE_CLASS") ?? "default",
  baseUrl: $env.get<string>("YT_BASE_URL") ?? "https://youtrack.example.com",
};

export default (): Chart => ({
  namespace: {
    apiVersion: "v1",
    kind: "Namespace",
    metadata: { name: values.namespace },
  },
  components: [
    YouTrackConfig({ baseUrl: values.baseUrl }),
    YouTrackStatefulSet({
      image: values.image,
      storageSize: values.storageSize,
      storageClass: values.storageClass,
    }),
    YouTrackService(),
    YouTrackIngress({ domain: values.domain }),
  ],
});
