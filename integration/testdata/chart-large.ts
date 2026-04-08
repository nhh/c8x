var components = [];
for (var i = 0; i < 20; i++) {
  components.push({
    apiVersion: "v1",
    kind: "ConfigMap",
    metadata: { name: "cm-" + i },
    data: { index: String(i) },
  });
}

export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-large-test" } },
  components: components,
});
