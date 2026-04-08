export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-file-test" } },
  components: [
    {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: { name: "nginx-config" },
      data: { "nginx.conf": $file.read("config.txt") },
    },
  ],
});
