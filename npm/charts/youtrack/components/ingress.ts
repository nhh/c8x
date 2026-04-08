import { Ingress } from "c8x";

export type IngressProps = {
  domain: string;
};

export default (props: IngressProps): Ingress => ({
  apiVersion: "networking.k8s.io/v1",
  kind: "Ingress",
  metadata: {
    name: "youtrack",
    annotations: {
      "kubernetes.io/ingress.class": "nginx",
      "nginx.ingress.kubernetes.io/proxy-body-size": "100m",
      "nginx.ingress.kubernetes.io/proxy-read-timeout": "3600",
      "nginx.ingress.kubernetes.io/proxy-send-timeout": "3600",
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
              backend: { service: { name: "youtrack", port: { number: 8080 } } },
            },
          ],
        },
      },
    ],
    tls: [{ hosts: [props.domain], secretName: "youtrack-tls" }],
  },
});
