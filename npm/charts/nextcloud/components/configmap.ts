import { ConfigMap } from "c8x";

export type NextcloudConfigProps = {
  domain: string;
  dbName: string;
  dbUser: string;
};

export default (props: NextcloudConfigProps): ConfigMap => ({
  apiVersion: "v1",
  kind: "ConfigMap",
  metadata: {
    name: "nextcloud-config",
  },
  data: {
    NEXTCLOUD_TRUSTED_DOMAINS: props.domain,
    NEXTCLOUD_OVERWRITEPROTOCOL: "https",
    NEXTCLOUD_OVERWRITECLIURL: `https://${props.domain}`,
    POSTGRES_HOST: "nextcloud-db",
    POSTGRES_DB: props.dbName,
    POSTGRES_USER: props.dbUser,
  },
});
