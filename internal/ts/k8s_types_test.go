package ts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubernetix/c8x/internal/k8s"
)

// k8sTypeTest defines a K8s resource type and a minimal valid chart using it.
type k8sTypeTest struct {
	name       string // test name
	kind       string // K8s kind field
	apiVersion string // K8s apiVersion field
	chartTS    string // TypeScript chart source
}

var k8sTypeTests = []k8sTypeTest{
	// --- core/v1 ---
	{
		name: "Namespace", kind: "Namespace", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "Namespace", metadata: { name: "extra-ns" } }
			]
		})`,
	},
	{
		name: "Service", kind: "Service", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "Service", metadata: { name: "my-svc" },
				  spec: { selector: { app: "test" }, ports: [{ port: 80 }], type: "ClusterIP" } }
			]
		})`,
	},
	{
		name: "ConfigMap", kind: "ConfigMap", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "ConfigMap", metadata: { name: "my-config" },
				  data: { KEY: "value", OTHER: "data" } }
			]
		})`,
	},
	{
		name: "Secret", kind: "Secret", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "Secret", metadata: { name: "my-secret" },
				  type: "Opaque", stringData: { password: "s3cret" } }
			]
		})`,
	},
	{
		name: "PersistentVolumeClaim", kind: "PersistentVolumeClaim", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "PersistentVolumeClaim", metadata: { name: "my-pvc" },
				  spec: { accessModes: ["ReadWriteOnce"],
				    resources: { requests: { storage: "5Gi" } } } }
			]
		})`,
	},
	{
		name: "PersistentVolume", kind: "PersistentVolume", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "PersistentVolume", metadata: { name: "my-pv" },
				  spec: { capacity: { storage: "10Gi" }, accessModes: ["ReadWriteOnce"],
				    hostPath: { path: "/data" } } }
			]
		})`,
	},
	{
		name: "ServiceAccount", kind: "ServiceAccount", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "ServiceAccount", metadata: { name: "my-sa" } }
			]
		})`,
	},
	{
		name: "Pod", kind: "Pod", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "Pod", metadata: { name: "my-pod" },
				  spec: { containers: [{ name: "app", image: "nginx" }] } }
			]
		})`,
	},
	{
		name: "Endpoints", kind: "Endpoints", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "Endpoints", metadata: { name: "my-ep" },
				  subsets: [{ addresses: [{ ip: "10.0.0.1" }],
				    ports: [{ port: 8080 }] }] }
			]
		})`,
	},
	{
		name: "LimitRange", kind: "LimitRange", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "LimitRange", metadata: { name: "my-lr" },
				  spec: { limits: [{ type: "Container",
				    default: { cpu: "500m", memory: "128Mi" } }] } }
			]
		})`,
	},
	{
		name: "ResourceQuota", kind: "ResourceQuota", apiVersion: "v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "v1", kind: "ResourceQuota", metadata: { name: "my-quota" },
				  spec: { hard: { pods: "10", "requests.cpu": "4" } } }
			]
		})`,
	},
	// --- networking/v1 ---
	{
		name: "Ingress", kind: "Ingress", apiVersion: "networking.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "networking.k8s.io/v1", kind: "Ingress", metadata: { name: "my-ing" },
				  spec: { rules: [{ host: "example.com",
				    http: { paths: [{ path: "/", pathType: "Prefix",
				      backend: { service: { name: "svc", port: { number: 80 } } } }] } }] } }
			]
		})`,
	},
	{
		name: "IngressClass", kind: "IngressClass", apiVersion: "networking.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "networking.k8s.io/v1", kind: "IngressClass",
				  metadata: { name: "nginx" },
				  spec: { controller: "k8s.io/ingress-nginx" } }
			]
		})`,
	},
	{
		name: "NetworkPolicy", kind: "NetworkPolicy", apiVersion: "networking.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "networking.k8s.io/v1", kind: "NetworkPolicy",
				  metadata: { name: "deny-all" },
				  spec: { podSelector: {}, policyTypes: ["Ingress", "Egress"] } }
			]
		})`,
	},
	// --- apps/v1 ---
	{
		name: "Deployment", kind: "Deployment", apiVersion: "apps/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "apps/v1", kind: "Deployment", metadata: { name: "my-deploy" },
				  spec: { replicas: 2, selector: { matchLabels: { app: "test" } },
				    template: { metadata: { labels: { app: "test" } },
				      spec: { containers: [{ name: "app", image: "nginx" }] } } } }
			]
		})`,
	},
	{
		name: "StatefulSet", kind: "StatefulSet", apiVersion: "apps/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "apps/v1", kind: "StatefulSet", metadata: { name: "my-sts" },
				  spec: { serviceName: "my-svc", replicas: 1,
				    selector: { matchLabels: { app: "db" } },
				    template: { metadata: { labels: { app: "db" } },
				      spec: { containers: [{ name: "db", image: "postgres:16" }] } } } }
			]
		})`,
	},
	{
		name: "DaemonSet", kind: "DaemonSet", apiVersion: "apps/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "apps/v1", kind: "DaemonSet", metadata: { name: "my-ds" },
				  spec: { selector: { matchLabels: { app: "agent" } },
				    template: { metadata: { labels: { app: "agent" } },
				      spec: { containers: [{ name: "agent", image: "fluentd" }] } } } }
			]
		})`,
	},
	{
		name: "ReplicaSet", kind: "ReplicaSet", apiVersion: "apps/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "apps/v1", kind: "ReplicaSet", metadata: { name: "my-rs" },
				  spec: { replicas: 3, selector: { matchLabels: { app: "web" } },
				    template: { metadata: { labels: { app: "web" } },
				      spec: { containers: [{ name: "web", image: "nginx" }] } } } }
			]
		})`,
	},
	// --- batch/v1 ---
	{
		name: "Job", kind: "Job", apiVersion: "batch/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "batch/v1", kind: "Job", metadata: { name: "my-job" },
				  spec: { template: {
				    spec: { containers: [{ name: "worker", image: "busybox",
				      command: ["echo", "hello"] }], restartPolicy: "Never" } } } }
			]
		})`,
	},
	{
		name: "CronJob", kind: "CronJob", apiVersion: "batch/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "batch/v1", kind: "CronJob", metadata: { name: "my-cron" },
				  spec: { schedule: "0 * * * *",
				    jobTemplate: { spec: { template: {
				      spec: { containers: [{ name: "cron", image: "busybox",
				        command: ["date"] }], restartPolicy: "OnFailure" } } } } } }
			]
		})`,
	},
	// --- rbac/v1 ---
	{
		name: "Role", kind: "Role", apiVersion: "rbac.authorization.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "rbac.authorization.k8s.io/v1", kind: "Role",
				  metadata: { name: "pod-reader" },
				  rules: [{ apiGroups: [""], resources: ["pods"], verbs: ["get", "list"] }] }
			]
		})`,
	},
	{
		name: "ClusterRole", kind: "ClusterRole", apiVersion: "rbac.authorization.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "rbac.authorization.k8s.io/v1", kind: "ClusterRole",
				  metadata: { name: "node-reader" },
				  rules: [{ apiGroups: [""], resources: ["nodes"], verbs: ["get", "list"] }] }
			]
		})`,
	},
	{
		name: "RoleBinding", kind: "RoleBinding", apiVersion: "rbac.authorization.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "rbac.authorization.k8s.io/v1", kind: "RoleBinding",
				  metadata: { name: "read-pods" },
				  roleRef: { apiGroup: "rbac.authorization.k8s.io", kind: "Role", name: "pod-reader" },
				  subjects: [{ kind: "ServiceAccount", name: "default", namespace: "test" }] }
			]
		})`,
	},
	{
		name: "ClusterRoleBinding", kind: "ClusterRoleBinding", apiVersion: "rbac.authorization.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "rbac.authorization.k8s.io/v1", kind: "ClusterRoleBinding",
				  metadata: { name: "read-nodes" },
				  roleRef: { apiGroup: "rbac.authorization.k8s.io", kind: "ClusterRole", name: "node-reader" },
				  subjects: [{ kind: "Group", name: "devs", apiGroup: "rbac.authorization.k8s.io" }] }
			]
		})`,
	},
	// --- autoscaling/v2 ---
	{
		name: "HorizontalPodAutoscaler", kind: "HorizontalPodAutoscaler", apiVersion: "autoscaling/v2",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "autoscaling/v2", kind: "HorizontalPodAutoscaler",
				  metadata: { name: "my-hpa" },
				  spec: { scaleTargetRef: { apiVersion: "apps/v1", kind: "Deployment", name: "my-deploy" },
				    minReplicas: 1, maxReplicas: 10,
				    metrics: [{ type: "Resource",
				      resource: { name: "cpu", target: { type: "Utilization", averageUtilization: 80 } } }] } }
			]
		})`,
	},
	// --- policy/v1 ---
	{
		name: "PodDisruptionBudget", kind: "PodDisruptionBudget", apiVersion: "policy/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "policy/v1", kind: "PodDisruptionBudget",
				  metadata: { name: "my-pdb" },
				  spec: { minAvailable: 1, selector: { matchLabels: { app: "web" } } } }
			]
		})`,
	},
	// --- storage/v1 ---
	{
		name: "StorageClass", kind: "StorageClass", apiVersion: "storage.k8s.io/v1",
		chartTS: `export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "test" } },
			components: [
				{ apiVersion: "storage.k8s.io/v1", kind: "StorageClass",
				  metadata: { name: "fast" },
				  provisioner: "kubernetes.io/aws-ebs",
				  parameters: { type: "gp3" },
				  reclaimPolicy: "Delete" }
			]
		})`,
	},
}

func TestK8sTypesFullPipeline(t *testing.T) {
	for _, tt := range k8sTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

			tsFile := filepath.Join(dir, "index.ts")
			os.WriteFile(tsFile, []byte(tt.chartTS), 0644)

			// Phase 1: esbuild transpile
			code, err := Load(tsFile, false)
			if err != nil {
				t.Fatalf("Load failed for %s: %v", tt.name, err)
			}

			if code == "" {
				t.Fatalf("Load produced empty output for %s", tt.name)
			}

			// Phase 2: Goja execute
			export, err := Run(code, tsFile)
			if err != nil {
				t.Fatalf("Run failed for %s: %v", tt.name, err)
			}

			// Phase 3: Verify structure
			if len(export.Components) == 0 {
				t.Fatalf("expected at least 1 component for %s", tt.name)
			}

			comp := export.Components[0]
			if comp == nil {
				t.Fatalf("component is nil for %s", tt.name)
			}

			if comp["kind"] != tt.kind {
				t.Fatalf("expected kind %q, got %q", tt.kind, comp["kind"])
			}

			if comp["apiVersion"] != tt.apiVersion {
				t.Fatalf("expected apiVersion %q, got %q", tt.apiVersion, comp["apiVersion"])
			}

			// Phase 4: Namespace patching
			if export.NamespaceName() != "test" {
				t.Fatalf("expected namespace name 'test', got %q", export.NamespaceName())
			}

			meta := comp.GetMetadata()
			if meta == nil {
				t.Fatalf("expected metadata on component for %s", tt.name)
			}

			if meta["namespace"] != "test" {
				t.Fatalf("expected namespace 'test' patched into %s, got %v", tt.name, meta["namespace"])
			}

			// Phase 5: PatchAndTransform → YAML
			chart := k8s.PatchAndTransform(export)

			if !strings.Contains(chart.Content, tt.kind) {
				t.Fatalf("expected YAML to contain kind %q for %s", tt.kind, tt.name)
			}

			if chart.Namespace == "" {
				t.Fatal("expected non-empty namespace YAML")
			}
		})
	}
}

// TestK8sTypesMultipleInOneChart verifies a chart with many different resource types.
func TestK8sTypesMultipleInOneChart(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"multi"}`), 0644)

	chart := `export default () => ({
		namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "multi" } },
		components: [
			{ apiVersion: "v1", kind: "ConfigMap", metadata: { name: "cfg" }, data: { k: "v" } },
			{ apiVersion: "v1", kind: "Secret", metadata: { name: "sec" }, type: "Opaque", stringData: { pw: "x" } },
			{ apiVersion: "v1", kind: "Service", metadata: { name: "svc" }, spec: { ports: [{ port: 80 }] } },
			{ apiVersion: "apps/v1", kind: "Deployment", metadata: { name: "app" },
			  spec: { replicas: 1, selector: { matchLabels: { app: "x" } },
			    template: { metadata: { labels: { app: "x" } },
			      spec: { containers: [{ name: "c", image: "nginx" }] } } } },
			{ apiVersion: "apps/v1", kind: "StatefulSet", metadata: { name: "db" },
			  spec: { serviceName: "db", replicas: 1, selector: { matchLabels: { app: "db" } },
			    template: { metadata: { labels: { app: "db" } },
			      spec: { containers: [{ name: "pg", image: "postgres" }] } } } },
			{ apiVersion: "batch/v1", kind: "CronJob", metadata: { name: "backup" },
			  spec: { schedule: "0 2 * * *",
			    jobTemplate: { spec: { template: {
			      spec: { containers: [{ name: "b", image: "busybox" }], restartPolicy: "Never" } } } } } },
			{ apiVersion: "networking.k8s.io/v1", kind: "Ingress", metadata: { name: "ing" },
			  spec: { rules: [{ host: "x.com", http: { paths: [{ path: "/", pathType: "Prefix",
			    backend: { service: { name: "svc", port: { number: 80 } } } }] } }] } },
		]
	})`

	tsFile := filepath.Join(dir, "index.ts")
	os.WriteFile(tsFile, []byte(chart), 0644)

	code, err := Load(tsFile, false)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	export, err := Run(code, tsFile)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(export.Components) != 7 {
		t.Fatalf("expected 7 components, got %d", len(export.Components))
	}

	expectedKinds := []string{"ConfigMap", "Secret", "Service", "Deployment", "StatefulSet", "CronJob", "Ingress"}
	for i, kind := range expectedKinds {
		if export.Components[i]["kind"] != kind {
			t.Fatalf("component %d: expected kind %q, got %q", i, kind, export.Components[i]["kind"])
		}
		if export.Components[i].GetMetadata()["namespace"] != "multi" {
			t.Fatalf("component %d (%s): namespace not patched", i, kind)
		}
	}

	result := k8s.PatchAndTransform(export)
	for _, kind := range expectedKinds {
		if !strings.Contains(result.Content, kind) {
			t.Fatalf("YAML missing kind %q", kind)
		}
	}

	fmt.Printf("Multi-chart YAML generated: %d bytes, %d documents\n",
		len(result.Content), strings.Count(result.Content, "---"))
}
