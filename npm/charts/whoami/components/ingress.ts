import {Ingress, IngressPath} from "c8x";

export type MyIngressProps = {
  domain: string;
};

const defaultBackend: IngressPath["backend"] = {
  service: {
    name: "whoami-svc",
    port: {
      number: 80,
    },
  },
};

const paths: IngressPath[] = [
  { path: "/", backend: defaultBackend, pathType: "ImplementationSpecific" },
];

export default (props: MyIngressProps): Ingress => ({
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
