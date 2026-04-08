import { Secret } from "c8x";

export type DbSecretProps = {
  dbName: string;
  dbUser: string;
  dbPassword: string;
};

export default (props: DbSecretProps): Secret => ({
  apiVersion: "v1",
  kind: "Secret",
  metadata: {
    name: "nextcloud-db-credentials",
  },
  type: "Opaque",
  stringData: {
    POSTGRES_DB: props.dbName,
    POSTGRES_USER: props.dbUser,
    POSTGRES_PASSWORD: props.dbPassword,
  },
});
