var isModern = $cluster.versionAtLeast("1.25");

export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-version-test" } },
  components: [
    {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: { name: "version-check" },
      data: { isModern: String(isModern) },
    },
  ],
});
