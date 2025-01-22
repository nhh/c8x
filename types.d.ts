// Todo add better types for right side (string|number)
declare const $env: {
  /** Parses `K8X_MY_TEST=abc` into abc */
  get<T>(name: string): T;

  /** Parses some env variables with the same prefix into a object
   * @example
   * 
   * $env.get("INGRESS_CLASS_ANNOTATIONS")
   * -----------
   * K8X_INGRESS_CLASS_ANNOTATIONS_KEY_1=nginx.ingress.kubernetes.io/app-root
   * K8X_INGRESS_CLASS_ANNOTATIONS_VALUE_1='/var/www/html'
   * -----------
   * K8X_INGRESS_CLASS_ANNOTATIONS_KEY_2=nginx.ingress.kubernetes.io/enable-cors
   * K8X_INGRESS_CLASS_ANNOTATIONS_VALUE_2=true
   * -----------
   * {
   *   "nginx.ingress.kubernetes.io/app-root": '/var/www/html',
   *   "nginx.ingress.kubernetes.io/enable-cors": true
   * }
   */
  getAsObject(prefix: string): k8x.Tuple;

  /** Parses a env variables as list
   * Consider these Variables:
   * K8X_MY_TEST_1=a
   * K8X_MY_TEST_2=b
   * K8X_MY_TEST_3=c
   * K8X_MY_TEST_4=d
   * K8X_MY_TEST_5=e
   * Will be parsed into
   * ["a", "b", "c", "d", "e"]
   */
  getAsList<T>(prefix: string): T[];
};

declare const $chart: {
  name: string;
  version: string;
  private: boolean;
  repository: {
    type: string;
    url: string;
  };
  files: string[];
  types: string;
  dependencies: k8x.Tuple;
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

declare namespace k8x {
  type Tuple = Record<string, string | number | boolean>;

  type Chart = {
    namespace?: Namespace;
    components: (
      | Namespace
      | Ingress
      | Deployment
      | Service
      | IngressClass
      | null
      | undefined
    )[];
  };

  // Definition für einen Pod (Kubernetes 1.31, TypeScript 5)

  // Definition für eine IngressClass (Kubernetes 1.31, TypeScript 5)
  type IngressClass = {
    apiVersion: "networking.k8s.io/v1";
    kind: "IngressClass";
    metadata: Metadata;
    spec: IngressClassSpec;
  };

  type IngressClassSpec = {
    controller: string;
    parameters?: IngressClassParameters;
  };

  type IngressClassParameters = {
    apiGroup?: string;
    kind: string;
    name: string;
    namespace?: string;
  };

  type Namespace = {
    apiVersion: "v1";
    kind: "Namespace";
    metadata: Metadata;
    spec?: NamespaceSpec;
    status?: NamespaceStatus;
  };

  type NamespaceSpec = {
    finalizers?: string[];
  };

  type NamespaceStatus = {
    phase?: "Active" | "Terminating";
  };

  type Pod = {
    apiVersion: "v1";
    kind: "Pod";
    metadata: Metadata;
    spec: PodSpec;
    status?: PodStatus;
  };

  type Metadata = {
    name: string;
    namespace?: string;
    labels?: Record<string, string | boolean | number>;
    annotations?: Record<string, string | boolean | number>;
    uid?: string;
    creationTimestamp?: string;
    ownerReferences?: OwnerReference[];
  };

  type OwnerReference = {
    apiVersion: string;
    kind: string;
    name: string;
    uid: string;
    controller?: boolean;
    blockOwnerDeletion?: boolean;
  };

  type PodSpec = {
    containers: Container[];
    restartPolicy?: "Always" | "OnFailure" | "Never";
    nodeName?: string;
    nodeSelector?: Tuple;
    serviceAccountName?: string;
    automountServiceAccountToken?: boolean;
  };

  type Container = {
    name: string;
    image: string;
    ports?: ContainerPort[];
    resources?: ResourceRequirements;
    env?: EnvVar[];
  };

  type ContainerPort = {
    containerPort: number;
    protocol?: "TCP" | "UDP" | "SCTP";
  };

  type ResourceRequirements = {
    limits?: Tuple;
    requests?: Tuple;
  };

  type EnvVar = {
    name: string;
    value?: string;
    valueFrom?: EnvVarSource;
  };

  type EnvVarSource = {
    fieldRef?: { fieldPath: string };
    resourceFieldRef?: { containerName?: string; resource: string };
  };

  type PodStatus = {
    phase: string;
    conditions?: PodCondition[];
    hostIP?: string;
    podIP?: string;
    startTime?: string;
  };

  type PodCondition = {
    type: string;
    status: string;
    lastProbeTime?: string;
    lastTransitionTime?: string;
  };

  // Definition für ein Deployment (Kubernetes 1.31, TypeScript 5)
  type Deployment = {
    apiVersion: "apps/v1";
    kind: "Deployment";
    metadata?: Metadata;
    spec?: DeploymentSpec;
    status?: DeploymentStatus;
  };

  type DeploymentSpec = {
    replicas?: number;
    selector: {
      matchLabels: Tuple;
    };
    template: PodTemplate;
    strategy?: DeploymentStrategy;
  };

  type PodTemplate = {
    metadata: Metadata;
    spec: PodSpec;
  };

  type DeploymentStrategy = {
    type: "Recreate" | "RollingUpdate";
    rollingUpdate?: RollingUpdateDeployment;
  };

  type RollingUpdateDeployment = {
    maxUnavailable?: number | string;
    maxSurge?: number | string;
  };

  type DeploymentStatus = {
    observedGeneration?: number;
    replicas: number;
    updatedReplicas?: number;
    readyReplicas?: number;
    availableReplicas?: number;
  };

  // Definition für einen Service (Kubernetes 1.31, TypeScript 5)
  type Service = {
    apiVersion: "v1";
    kind: "Service";
    metadata: Metadata;
    spec: ServiceSpec;
    status?: ServiceStatus;
  };

  type ServiceSpec = {
    type: "ClusterIP" | "NodePort" | "LoadBalancer";
    ports: ServicePort[];
    selector?: Tuple;
    clusterIP?: string;
    externalIPs?: string[];
    sessionAffinity?: "None" | "ClientIP";
  };

  type ServicePort = {
    protocol?: "TCP" | "UDP" | "SCTP";
    port: number;
    targetPort?: number | string;
    nodePort?: number;
  };

  type ServiceStatus = {
    loadBalancer?: {
      ingress?: { ip: string; hostname?: string }[];
    };
  };

  // Definition für eine ConfigMap (Kubernetes 1.31, TypeScript 5)
  type ConfigMap = {
    apiVersion: "v1";
    kind: "ConfigMap";
    metadata: Metadata;
    data: Tuple;
    binaryData?: Tuple;
  };

  // Definition für ein Secret (Kubernetes 1.31, TypeScript 5)
  type Secret = {
    apiVersion: "v1";
    kind: "Secret";
    metadata: Metadata;
    data: Tuple;
    stringData?: Tuple;
    type: string;
  };

  // Definition für einen Ingress (Kubernetes 1.31, TypeScript 5)
  type Ingress = {
    apiVersion: "networking.k8s.io/v1";
    kind: "Ingress";
    metadata?: Metadata;
    spec: IngressSpec;
    status?: IngressStatus;
  };

  type IngressSpec = {
    rules: IngressRule[];
    tls?: IngressTLS[];
  };

  type IngressRule = {
    host: string;
    http: {
      paths: IngressPath[];
    };
  };

  type IngressPath = {
    path: string;
    pathType: "Prefix" | "Exact" | "ImplementationSpecific";
    backend: {
      service: {
        name: string;
        port: {
          number: number;
        };
      };
    };
  };

  type IngressTLS = {
    hosts: string[];
    secretName: string;
  };

  type IngressStatus = {
    loadBalancer?: {
      ingress?: { ip: string; hostname?: string }[];
    };
  };

  // Definition für ein PersistentVolumeClaim (PVC) (Kubernetes 1.31, TypeScript 5)
  type PersistentVolumeClaim = {
    apiVersion: "v1";
    kind: "PersistentVolumeClaim";
    metadata: Metadata;
    spec: PersistentVolumeClaimSpec;
    status?: PersistentVolumeClaimStatus;
  };

  type PersistentVolumeClaimSpec = {
    accessModes: ("ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany")[];
    resources: {
      requests: {
        storage: string;
      };
    };
    storageClassName?: string;
    volumeMode?: "Filesystem" | "Block";
  };

  type PersistentVolumeClaimStatus = {
    phase: string;
    capacity?: Tuple;
    accessModes?: string[];
  };
}
