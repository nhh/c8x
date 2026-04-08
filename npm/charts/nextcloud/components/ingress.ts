import { Ingress } from "c8x";

export type IngressProps = {
  domain: string;
};

export default (props: IngressProps): Ingress => ({
  apiVersion: "networking.k8s.io/v1",
  kind: "Ingress",
  metadata: {
    name: "nextcloud",
    annotations: {
      "kubernetes.io/ingress.class": "nginx",
      "nginx.ingress.kubernetes.io/proxy-body-size": "512m",
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
              backend: {
                service: {
                  name: "nextcloud",
                  port: { number: 80 },
                },
              },
            },
          ],
        },
      },
    ],
    tls: [
      {
        hosts: [props.domain],
        secretName: "nextcloud-tls",
      },
    ],
  },
});
