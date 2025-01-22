// @ts-check

/**
 * @typedef {Object} MyIngressProps - creates a new type named 'SpecialType'
 * @prop {string} name - a string property of SpecialType
 * @prop {string?} appRoot - a number property of SpecialType
 * @prop {Record<string, string>} annotations - a number property of SpecialType
 * @prop {string[]} additionalPaths - an optional number property of SpecialType
 */

/** @type {IngressPath["backend"]} */
const defaultBackend = {
  service: {
    name: 'super-duper-service',
    port: {
      number: 8080,
    },
  },
};

/** @type {(additionalPaths?: string[]) => IngressPath[]} */
function generateIngressPaths(additionalPaths = []) {

  /** @type {IngressPath[]} */
  const paths = [
    { path: "/", backend: defaultBackend, pathType: "ImplementationSpecific" },
  ];

  for (const path of additionalPaths) {
    paths.push({
      path: path,
      backend: defaultBackend,
      pathType: "ImplementationSpecific",
    });
  }

  return paths;
}

/** @type {(props: MyIngressProps) => Ingress} */
export default (props) => ({
  apiVersion: "networking.k8s.io/v1",
  kind: "Ingress",
  spec: {
    rules: [{ host: "pfusch.dev", http: { paths: generateIngressPaths() } }],
  },
  metadata: {
    name: props.name,
    annotations: {
      "nginx.ingress.kubernetes.io/app-root": "/var/www/html",
      "nginx.ingress.kubernetes.io/enable-cors": "true",
      "nginx.ingress.kubernetes.io/cors-allow-origin": "https://example.com",
      ...props.annotations,
    },
  },
});
