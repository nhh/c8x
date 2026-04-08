import { Secret } from "c8x";

export type GrafanaSecretProps = {
  adminPassword: string;
};

export default (props: GrafanaSecretProps): Secret => ({
  apiVersion: "v1",
  kind: "Secret",
  metadata: { name: "grafana-credentials" },
  type: "Opaque",
  stringData: {
    GF_SECURITY_ADMIN_USER: "admin",
    GF_SECURITY_ADMIN_PASSWORD: props.adminPassword,
  },
});
