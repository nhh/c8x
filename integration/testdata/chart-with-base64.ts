export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-b64-test" } },
  components: [
    {
      apiVersion: "v1",
      kind: "Secret",
      metadata: { name: "encoded-secret" },
      type: "Opaque",
      data: { password: $base64.encode("super-secret-123") },
    },
  ],
});
