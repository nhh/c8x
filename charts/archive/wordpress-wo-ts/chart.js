// @ts-check

/** @typedef {import('./components/ingress').MyIngressProps} MyIngressProps */
import MyIngress from "./components/ingress";

/** @type {{ ingress: MyIngressProps  }} */
const values = {
  ingress: {
    name: $env["INGRESS_NAME"] ?? "my-ingress",
    appRoot: $env["INGRESS_NAME"] ?? "/var/www/html",
    additionalPaths:
      Object.keys($env)
        .filter((key) => key.startsWith("ADDITIONAL_INGRESS_PATH"))
        .map((key) => $env[key]) ?? [],
  },
};

// Values which are configurable
/** @type { () => {name: string, namespace: 'default', ingresses: IngressPath[]} } */
export default () => ({
  name: "default",
  namespace: "default",
  ingresses: [MyIngress(values.ingress)],
});
