var config = $yaml.parse($file.read("prometheus.yml"));
config.global = { scrape_interval: "30s" };

export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-yaml-test" } },
  components: [
    {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: { name: "prometheus-config" },
      data: { "prometheus.yml": $yaml.stringify(config) },
    },
  ],
});
