export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-hash-test" } },
  components: [
    {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: {
        name: "hashed-config",
        annotations: { "c8x/config-hash": $hash.sha256("deterministic-input") },
      },
      data: { key: "value" },
    },
  ],
});
