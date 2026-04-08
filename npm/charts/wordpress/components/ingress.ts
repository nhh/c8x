import { Ingress } from "c8x";

export type IngressProps = {
  domain: string;
};

export default (props: IngressProps): Ingress => ({
  apiVersion: "networking.k8s.io/v1",
  kind: "Ingress",
  metadata: {
    name: "wordpress",
    annotations: {
      "kubernetes.io/ingress.class": "nginx",
      "nginx.ingress.kubernetes.io/proxy-body-size": "64m",
    },
  },
  spec: {
    rules: [
      {
        host: props.domain,
        http: {
          paths: [
            {
              path: "/",
              pathType: "Prefix",
              backend: { service: { name: "wordpress", port: { number: 80 } } },
            },
          ],
        },
      },
    ],
    tls: [{ hosts: [props.domain], secretName: "wordpress-tls" }],
  },
});
