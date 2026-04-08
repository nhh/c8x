import { Deployment } from "c8x";

export type NextcloudDeploymentProps = {
  replicas: number;
  image: string;
};

export default (props: NextcloudDeploymentProps): Deployment => ({
  apiVersion: "apps/v1",
  kind: "Deployment",
  metadata: {
    name: "nextcloud",
  },
  spec: {
    replicas: props.replicas,
    selector: { matchLabels: { app: "nextcloud" } },
    strategy: { type: "Recreate" },
    template: {
      metadata: { labels: { app: "nextcloud" } },
      spec: {
        containers: [
          {
            name: "nextcloud",
            image: props.image,
            ports: [{ containerPort: 80, protocol: "TCP" }],
            envFrom: [
              { configMapRef: { name: "nextcloud-config" } },
              { secretRef: { name: "nextcloud-db-credentials" } },
            ],
            volumeMounts: [
              { name: "nextcloud-data", mountPath: "/var/www/html" },
            ],
          },
        ],
        volumes: [
          {
            name: "nextcloud-data",
            persistentVolumeClaim: { claimName: "nextcloud-data" },
          },
        ],
      },
    },
  },
});
