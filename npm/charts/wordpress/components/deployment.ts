import { Deployment } from "c8x";

export type WordpressDeploymentProps = {
  replicas: number;
  image: string;
  domain: string;
};

export default (props: WordpressDeploymentProps): Deployment => ({
  apiVersion: "apps/v1",
  kind: "Deployment",
  metadata: { name: "wordpress" },
  spec: {
    replicas: props.replicas,
    selector: { matchLabels: { app: "wordpress" } },
    strategy: { type: "Recreate" },
    template: {
      metadata: { labels: { app: "wordpress" } },
      spec: {
        containers: [
          {
            name: "wordpress",
            image: props.image,
            ports: [{ containerPort: 80, protocol: "TCP" }],
            env: [
              { name: "WORDPRESS_DB_HOST", value: "wordpress-db" },
              { name: "WORDPRESS_DB_NAME", valueFrom: { secretKeyRef: { name: "wordpress-db-credentials", key: "MYSQL_DATABASE" } } },
              { name: "WORDPRESS_DB_USER", valueFrom: { secretKeyRef: { name: "wordpress-db-credentials", key: "MYSQL_USER" } } },
              { name: "WORDPRESS_DB_PASSWORD", valueFrom: { secretKeyRef: { name: "wordpress-db-credentials", key: "MYSQL_PASSWORD" } } },
              { name: "WORDPRESS_CONFIG_EXTRA", value: `define('WP_HOME','https://${props.domain}');define('WP_SITEURL','https://${props.domain}');` },
            ],
            volumeMounts: [
              { name: "wordpress-data", mountPath: "/var/www/html" },
            ],
          },
        ],
        volumes: [
          {
            name: "wordpress-data",
            persistentVolumeClaim: { claimName: "wordpress-data" },
          },
        ],
      },
    },
  },
});
