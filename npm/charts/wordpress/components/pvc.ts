import { PersistentVolumeClaim } from "c8x";

export type PvcProps = {
  storageSize: string;
  storageClass: string;
};

export default (props: PvcProps): PersistentVolumeClaim => ({
  apiVersion: "v1",
  kind: "PersistentVolumeClaim",
  metadata: { name: "wordpress-data" },
  spec: {
    accessModes: ["ReadWriteOnce"],
    storageClassName: props.storageClass,
    resources: { requests: { storage: props.storageSize } },
  },
});
