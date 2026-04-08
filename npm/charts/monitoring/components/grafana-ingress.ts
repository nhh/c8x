import { Ingress } from "c8x";

export type GrafanaIngressProps = {
  domain: string;
};

export default (props: GrafanaIngressProps): Ingress => ({
  apiVersion: "networking.k8s.io/v1",
  kind: "Ingress",
  metadata: {
    name: "grafana",
    annotations: { "kubernetes.io/ingress.class": "nginx" },
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
              backend: { service: { name: "grafana", port: { number: 3000 } } },
            },
          ],
        },
      },
    ],
    tls: [{ hosts: [props.domain], secretName: "grafana-tls" }],
  },
});
