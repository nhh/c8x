import { PersistentVolumeClaim } from "c8x";

export type GrafanaPvcProps = {
  storageSize: string;
  storageClass: string;
};

export default (props: GrafanaPvcProps): PersistentVolumeClaim => ({
  apiVersion: "v1",
  kind: "PersistentVolumeClaim",
  metadata: { name: "grafana-data" },
  spec: {
    accessModes: ["ReadWriteOnce"],
    storageClassName: props.storageClass,
    resources: { requests: { storage: props.storageSize } },
  },
});
