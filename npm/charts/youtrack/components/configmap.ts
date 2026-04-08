import { ConfigMap } from "c8x";

export type YouTrackConfigProps = {
  baseUrl: string;
};

export default (props: YouTrackConfigProps): ConfigMap => ({
  apiVersion: "v1",
  kind: "ConfigMap",
  metadata: { name: "youtrack-config" },
  data: {
    YOUTRACK_BASE_URL: props.baseUrl,
  },
});
