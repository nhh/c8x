import {Namespace, Service} from "kubernetes-types/core/v1";
import {Ingress, IngressClass} from "kubernetes-types/networking/v1";
import {Deployment} from "kubernetes-types/apps/v1";

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
}

export type Chart = {
  namespace?: Namespace;
  components: (
    | Namespace
    | Ingress
    | Deployment
    | Service
    | IngressClass
    )[];
};
