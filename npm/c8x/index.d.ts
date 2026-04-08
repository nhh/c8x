export {
  Namespace,
  Service,
  ConfigMap,
  Secret,
  PersistentVolumeClaim,
  PersistentVolume,
  ServiceAccount,
  Pod,
  Endpoints,
  LimitRange,
  ResourceQuota,
} from "kubernetes-types/core/v1";
export {
  Ingress,
  IngressClass,
  IngressRule,
  HTTPIngressPath,
  NetworkPolicy,
} from "kubernetes-types/networking/v1";
export {
  Deployment,
  StatefulSet,
  DaemonSet,
  ReplicaSet,
} from "kubernetes-types/apps/v1";
export { Job, CronJob } from "kubernetes-types/batch/v1";
export {
  Role,
  ClusterRole,
  RoleBinding,
  ClusterRoleBinding,
} from "kubernetes-types/rbac/v1";
export { HorizontalPodAutoscaler } from "kubernetes-types/autoscaling/v2";
export { PodDisruptionBudget } from "kubernetes-types/policy/v1";
export { StorageClass } from "kubernetes-types/storage/v1";

import { Namespace, Service, ConfigMap, Secret, PersistentVolumeClaim, PersistentVolume, ServiceAccount, Pod, Endpoints, LimitRange, ResourceQuota } from "kubernetes-types/core/v1";
import { Ingress, IngressClass, NetworkPolicy } from "kubernetes-types/networking/v1";
import { Deployment, StatefulSet, DaemonSet, ReplicaSet } from "kubernetes-types/apps/v1";
import { Job, CronJob } from "kubernetes-types/batch/v1";
import { Role, ClusterRole, RoleBinding, ClusterRoleBinding } from "kubernetes-types/rbac/v1";
import { HorizontalPodAutoscaler } from "kubernetes-types/autoscaling/v2";
import { PodDisruptionBudget } from "kubernetes-types/policy/v1";
import { StorageClass } from "kubernetes-types/storage/v1";

import { HTTPIngressPath } from "kubernetes-types/networking/v1";
export type IngressPath = HTTPIngressPath;

export type Tuple = Record<string, string | number | boolean>;

declare global {
  const $env: {
    /** Parses `C8X_MY_TEST=abc` into abc */
    get<T>(name: string): T;

    /** Parses some env variables with the same prefix into a object
     * @example
     *
     * $env.get("INGRESS_CLASS_ANNOTATIONS")
     * -----------
     * C8X_INGRESS_CLASS_ANNOTATIONS_KEY_1=nginx.ingress.kubernetes.io/app-root
     * C8X_INGRESS_CLASS_ANNOTATIONS_VALUE_1='/var/www/html'
     * -----------
     * C8X_INGRESS_CLASS_ANNOTATIONS_KEY_2=nginx.ingress.kubernetes.io/enable-cors
     * C8X_INGRESS_CLASS_ANNOTATIONS_VALUE_2=true
     * -----------
     * {
     *   "nginx.ingress.kubernetes.io/app-root": '/var/www/html',
     *   "nginx.ingress.kubernetes.io/enable-cors": true
     * }
     */
    getAsObject(prefix: string): Tuple;

    /** Parses a env variables as list
     * Consider these Variables:
     * C8X_MY_TEST_1=a
     * C8X_MY_TEST_2=b
     * C8X_MY_TEST_3=c
     * C8X_MY_TEST_4=d
     * C8X_MY_TEST_5=e
     * Will be parsed into
     * ["a", "b", "c", "d", "e"]
     */
    getAsList<T>(prefix: string): T[];
  };
  const $chart: {
    name: string;
    version: string;
    private: boolean;
    repository: {
      type: string;
      url: string;
    };
    files: string[];
    types: string;
    dependencies: Tuple;
    appVersion: string;
    kubeVersion: string;
    type: string;
    keywords: string[];
    home: string;
    maintainers: string[];
    icon: string;
    deprecated: boolean;
    annotations: string[];
  };

  const $base64: {
    /** Encode a string to base64 */
    encode(input: string): string;
    /** Decode a base64 string */
    decode(input: string): string;
  };

  const $hash: {
    /** SHA-256 hash of a string (hex-encoded) */
    sha256(input: string): string;
    /** MD5 hash of a string (hex-encoded) */
    md5(input: string): string;
  };

  const $log: {
    /** Log an info message during chart compilation */
    info(message: string): void;
    /** Log a warning during chart compilation */
    warn(message: string): void;
    /** Log an error during chart compilation */
    error(message: string): void;
  };

  /**
   * Assert a condition. Throws an error with the given message if the condition is falsy.
   * @example
   * $assert(password !== "changeme", "Set a real DB_PASSWORD");
   * $assert(replicas >= 1 && replicas <= 20, "REPLICAS must be 1-20");
   */
  const $assert: (condition: unknown, message: string) => void;

  const $file: {
    /** Read a file relative to the chart directory */
    read(path: string): string;
    /** Check if a file exists relative to the chart directory */
    exists(path: string): boolean;
  };

  interface HttpResponse {
    status: number;
    body: string;
    headers: Record<string, string>;
  }

  interface HttpOptions {
    headers?: Record<string, string>;
  }

  const $yaml: {
    /** Parse a YAML string into a JavaScript object */
    parse<T = unknown>(input: string): T;
    /** Stringify a JavaScript object into a YAML string */
    stringify(input: unknown): string;
  };

  const $cluster: {
    /** Returns the Kubernetes server version (e.g. "1.31") */
    version(): string;
    /** Returns true if the cluster version is at least the given version */
    versionAtLeast(version: string): boolean;
    /** Returns the number of nodes in the cluster */
    nodeCount(): number;
    /** Returns true if the given API group/version is available */
    apiAvailable(apiVersion: string): boolean;
    /** Returns true if the given CRD exists in the cluster */
    crdExists(name: string): boolean;
    /** Returns true if the given resource exists */
    exists(apiVersion: string, kind: string, namespace: string, name: string): boolean;
    /** Lists resources of the given kind, optionally in a namespace */
    list<T = unknown>(kind: string, namespace?: string): T[];
  };

  const $http: {
    /** Perform a GET request, returns full response */
    get(url: string, options?: HttpOptions): HttpResponse;
    /** Perform a GET request, returns body as string */
    getText(url: string, options?: HttpOptions): string;
    /** Perform a GET request, returns parsed JSON */
    getJSON<T = unknown>(url: string, options?: HttpOptions): T;
    /** Perform a POST request with a string body */
    post(url: string, body: string, options?: HttpOptions): HttpResponse;
    /** Perform a POST request with a JSON body, returns parsed JSON */
    postJSON<T = unknown>(url: string, body: unknown, options?: HttpOptions): T;
  };
}

export type Chart = {
  namespace?: Namespace;
  components: (
    | Namespace
    | Service
    | ConfigMap
    | Secret
    | PersistentVolumeClaim
    | PersistentVolume
    | ServiceAccount
    | Pod
    | Endpoints
    | LimitRange
    | ResourceQuota
    | Ingress
    | IngressClass
    | NetworkPolicy
    | Deployment
    | StatefulSet
    | DaemonSet
    | ReplicaSet
    | Job
    | CronJob
    | Role
    | ClusterRole
    | RoleBinding
    | ClusterRoleBinding
    | HorizontalPodAutoscaler
    | PodDisruptionBudget
    | StorageClass
  )[];
};
