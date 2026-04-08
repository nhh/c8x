export default () => ({
  namespace: {
    apiVersion: "v1",
    kind: "Namespace",
    metadata: { name: "c8x-integration-test" },
  },
  components: [
    {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: { name: "test-config" },
      data: { key: "value", version: "v1" },
    },
    {
      apiVersion: "v1",
      kind: "Service",
      metadata: { name: "test-svc" },
      spec: {
        selector: { app: "test" },
        ports: [{ port: 80, targetPort: 80 }],
        type: "ClusterIP",
      },
    },
  ],
});
