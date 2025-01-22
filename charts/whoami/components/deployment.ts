export type MyDeploymentProps = {
  replicas: number;
};

export default (props: MyDeploymentProps): k8x.Deployment => ({
  apiVersion: "apps/v1",
  kind: "Deployment",
  spec: {
    replicas: props.replicas,
    selector: { matchLabels: { app: "whoami" } },
    template: {
      metadata: { name: "whoami", labels: { app: "whoami" } },
      spec: {
        containers: [
          {
            image: "traefik/whoami",
            name: "whoami",
            ports: [{ containerPort: 80, protocol: "TCP" }],
          },
        ],
      },
    },
  },
  metadata: {
    name: 'whoami-deployment',
  },
});
