$assert(false, "This chart intentionally fails validation");

export default () => ({
  namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "should-not-deploy" } },
  components: [],
});
