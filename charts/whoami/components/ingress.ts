export type MyIngressProps = {
  domain: string;
};

const defaultBackend: k8x.IngressPath["backend"] = {
  service: {
    name: "whoami-svc",
    port: {
      number: 80,
    },
  },
};

const paths: k8x.IngressPath[] = [
  { path: "/", backend: defaultBackend, pathType: "ImplementationSpecific" },
];

export default (props: MyIngressProps): k8x.Ingress => ({
  apiVersion: "networking.k8s.io/v1",
  kind: "Ingress",
  spec: {
    rules: [{ host: props.domain, http: { paths } }],
  },
  metadata: {
    name: "whoami-ingress",
    annotations: { "kubernetes.io/ingress.class": "nginx" },
  },
});
