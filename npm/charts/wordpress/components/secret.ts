import { Secret } from "c8x";

export type DbSecretProps = {
  dbName: string;
  dbUser: string;
  dbPassword: string;
  dbRootPassword: string;
};

export default (props: DbSecretProps): Secret => ({
  apiVersion: "v1",
  kind: "Secret",
  metadata: { name: "wordpress-db-credentials" },
  type: "Opaque",
  stringData: {
    MYSQL_DATABASE: props.dbName,
    MYSQL_USER: props.dbUser,
    MYSQL_PASSWORD: props.dbPassword,
    MYSQL_ROOT_PASSWORD: props.dbRootPassword,
  },
});
