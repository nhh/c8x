//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kubernetix/c8x/internal/k8s"
	"github.com/kubernetix/c8x/internal/ts"
)

var (
	testClient    *k8s.Client
	testdataDir   string
	chartPath     string
	chartV2Path   string
)

func TestMain(m *testing.M) {
	// Find testdata directory
	_, filename, _, _ := runtime.Caller(0)
	testdataDir = filepath.Join(filepath.Dir(filename), "testdata")
	chartPath = filepath.Join(testdataDir, "chart.ts")
	chartV2Path = filepath.Join(testdataDir, "chart-v2.ts")

	// Create KinD cluster
	kubeconfigPath, err := CreateTestCluster()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create test cluster: %v\n", err)
		os.Exit(1)
	}
	os.Setenv("KUBECONFIG", kubeconfigPath)

	// Create client
	testClient, err = k8s.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create K8s client: %v\n", err)
		DeleteTestCluster()
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	DeleteTestCluster()
	os.Remove(kubeconfigPath)
	os.Exit(code)
}

// ==================== Client Tests ====================

func TestClientServerVersion(t *testing.T) {
	version, err := testClient.ServerVersion()
	if err != nil {
		t.Fatalf("ServerVersion failed: %v", err)
	}
	if !strings.Contains(version, ".") {
		t.Fatalf("expected version like '1.31', got %q", version)
	}
	t.Logf("Cluster version: %s", version)
}

func TestClientNodeCount(t *testing.T) {
	count, err := testClient.NodeCount()
	if err != nil {
		t.Fatalf("NodeCount failed: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least 1 node, got %d", count)
	}
	t.Logf("Node count: %d", count)
}

func TestClientAPIAvailable(t *testing.T) {
	if !testClient.APIAvailable("apps/v1") {
		t.Fatal("expected apps/v1 to be available")
	}
	if !testClient.APIAvailable("v1") {
		t.Fatal("expected core v1 to be available")
	}
	if testClient.APIAvailable("nonexistent.example.com/v99") {
		t.Fatal("expected nonexistent API to not be available")
	}
}

func TestClientApplyAndDelete(t *testing.T) {
	yaml := `apiVersion: v1
kind: Namespace
metadata:
  name: c8x-apply-test
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: c8x-apply-test
data:
  hello: world`

	// Apply
	output, err := testClient.Apply([]byte(yaml))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	t.Logf("Apply output: %s", output)

	// Verify exists
	if !testClient.ResourceExists("ConfigMap", "c8x-apply-test", "test-cm") {
		t.Fatal("ConfigMap should exist after apply")
	}

	// Delete
	deleteOutput, err := testClient.Delete([]byte(yaml))
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	t.Logf("Delete output: %s", deleteOutput)

	// Wait briefly for deletion
	time.Sleep(500 * time.Millisecond)

	// Verify gone
	if testClient.ResourceExists("ConfigMap", "c8x-apply-test", "test-cm") {
		t.Fatal("ConfigMap should not exist after delete")
	}
}

func TestClientListResources(t *testing.T) {
	ns := "c8x-list-test"

	// Create namespace
	testClient.Apply([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))

	// Create 3 ConfigMaps
	for i := 1; i <= 3; i++ {
		yaml := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-%d
  namespace: %s
data:
  index: "%d"`, i, ns, i)
		testClient.Apply([]byte(yaml))
	}

	items, err := testClient.ListResources("ConfigMap", ns)
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	// KinD creates a kube-root-ca.crt ConfigMap automatically, so >= 3
	if len(items) < 3 {
		t.Fatalf("expected at least 3 ConfigMaps, got %d", len(items))
	}

	// Cleanup
	testClient.Delete([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))
}

// ==================== Release CRUD Tests ====================

func TestReleaseCreateAndRead(t *testing.T) {
	ns := "c8x-release-test"
	testClient.Apply([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))
	defer testClient.Delete([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))

	release := &k8s.Release{
		Name:          "test-release",
		Revision:      1,
		Status:        k8s.StatusDeployed,
		Namespace:     ns,
		Manifest:      "test-manifest",
		DeployedAt:    time.Now(),
		Trigger:       k8s.TriggerManual,
		Resources:     []string{"ConfigMap/test"},
		ResourceCount: 1,
		Runtime:       k8s.CollectRuntime(),
		Deployer:      k8s.CollectDeployer(),
	}

	// Save
	if err := testClient.SaveRelease(release); err != nil {
		t.Fatalf("SaveRelease failed: %v", err)
	}

	// Read
	current, err := testClient.GetCurrentRelease(ns, "test-release")
	if err != nil {
		t.Fatalf("GetCurrentRelease failed: %v", err)
	}
	if current == nil {
		t.Fatal("expected release, got nil")
	}
	if current.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", current.Revision)
	}
	if current.Status != k8s.StatusDeployed {
		t.Fatalf("expected deployed, got %s", current.Status)
	}
	if current.Trigger != k8s.TriggerManual {
		t.Fatalf("expected manual, got %s", current.Trigger)
	}
}

func TestReleaseUpdateStatus(t *testing.T) {
	ns := "c8x-status-test"
	testClient.Apply([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))
	defer testClient.Delete([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))

	release := &k8s.Release{
		Name: "status-test", Revision: 1, Status: k8s.StatusDeployed,
		Namespace: ns, Manifest: "test", DeployedAt: time.Now(), Trigger: k8s.TriggerManual,
	}
	testClient.SaveRelease(release)

	// Update to superseded
	err := testClient.UpdateReleaseStatus(release, k8s.StatusSuperseded)
	if err != nil {
		t.Fatalf("UpdateReleaseStatus failed: %v", err)
	}

	// Verify: GetCurrentRelease should return nil (no deployed)
	current, _ := testClient.GetCurrentRelease(ns, "status-test")
	if current != nil {
		t.Fatal("expected nil after superseding")
	}

	// But GetRelease by revision should work
	r, err := testClient.GetRelease(ns, "status-test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != k8s.StatusSuperseded {
		t.Fatalf("expected superseded, got %s", r.Status)
	}
}

func TestReleaseListAndDeleteOld(t *testing.T) {
	ns := "c8x-gc-test"
	testClient.Apply([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))
	defer testClient.Delete([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))

	// Create 5 revisions
	for i := 1; i <= 5; i++ {
		r := &k8s.Release{
			Name: "gc-test", Revision: i, Status: k8s.StatusSuperseded,
			Namespace: ns, Manifest: fmt.Sprintf("manifest-v%d", i),
			DeployedAt: time.Now(), Trigger: k8s.TriggerManual,
		}
		if i == 5 {
			r.Status = k8s.StatusDeployed
		}
		testClient.SaveRelease(r)
	}

	// List all
	releases, err := testClient.ListReleases(ns, "gc-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 5 {
		t.Fatalf("expected 5 releases, got %d", len(releases))
	}

	// Delete old, keep 3
	testClient.DeleteOldRevisions(ns, "gc-test", 3)

	releases, _ = testClient.ListReleases(ns, "gc-test")
	if len(releases) != 3 {
		t.Fatalf("expected 3 releases after GC, got %d", len(releases))
	}
	// Should keep v3, v4, v5
	if releases[0].Revision != 3 {
		t.Fatalf("expected oldest kept = v3, got v%d", releases[0].Revision)
	}
}

// ==================== Full Pipeline Tests ====================

func compileAndApply(t *testing.T, chartFile string) k8s.Chart {
	t.Helper()
	code, err := ts.Load(chartFile, false)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	export, err := ts.Run(code, chartFile, ts.AllPermissions())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	chart := k8s.PatchAndTransform(export)

	if err := k8s.ApplyChart(testClient, chart); err != nil {
		t.Fatalf("ApplyChart failed: %v", err)
	}

	return chart
}

func TestFullLifecycle(t *testing.T) {
	ns := "c8x-integration-test"
	name := ns

	// --- Install ---
	chart := compileAndApply(t, chartPath)

	release := &k8s.Release{
		Name: name, Revision: 1, Status: k8s.StatusDeployed,
		Namespace: ns, Manifest: chart.Combined(), DeployedAt: time.Now(),
		Trigger: k8s.TriggerManual, Resources: k8s.ExtractResources(chart.Combined()),
		ResourceCount: len(k8s.ExtractResources(chart.Combined())),
		Runtime: k8s.CollectRuntime(), Deployer: k8s.CollectDeployer(),
	}
	if err := testClient.SaveRelease(release); err != nil {
		t.Fatalf("SaveRelease failed: %v", err)
	}
	t.Log("Install: OK")

	// Verify resources exist
	if !testClient.ResourceExists("ConfigMap", ns, "test-config") {
		t.Fatal("ConfigMap test-config should exist")
	}
	if !testClient.ResourceExists("Service", ns, "test-svc") {
		t.Fatal("Service test-svc should exist")
	}

	// --- Status ---
	current, err := testClient.GetCurrentRelease(ns, name)
	if err != nil || current == nil {
		t.Fatalf("GetCurrentRelease failed: %v", err)
	}
	if current.Revision != 1 || current.Status != k8s.StatusDeployed {
		t.Fatalf("unexpected status: rev=%d status=%s", current.Revision, current.Status)
	}
	t.Log("Status: OK")

	// --- Upgrade ---
	chartV2 := compileAndApply(t, chartV2Path)

	testClient.UpdateReleaseStatus(current, k8s.StatusSuperseded)
	releaseV2 := &k8s.Release{
		Name: name, Revision: 2, Status: k8s.StatusDeployed,
		Namespace: ns, Manifest: chartV2.Combined(), DeployedAt: time.Now(),
		Trigger: k8s.TriggerManual, Resources: k8s.ExtractResources(chartV2.Combined()),
		ResourceCount: len(k8s.ExtractResources(chartV2.Combined())),
		Runtime: k8s.CollectRuntime(), Deployer: k8s.CollectDeployer(),
	}
	testClient.SaveRelease(releaseV2)
	t.Log("Upgrade: OK")

	// Verify extra-config exists (added in v2)
	if !testClient.ResourceExists("ConfigMap", ns, "extra-config") {
		t.Fatal("ConfigMap extra-config should exist after upgrade")
	}

	// --- History ---
	releases, _ := testClient.ListReleases(ns, name)
	if len(releases) != 2 {
		t.Fatalf("expected 2 revisions in history, got %d", len(releases))
	}
	if releases[0].Status != k8s.StatusSuperseded || releases[1].Status != k8s.StatusDeployed {
		t.Fatal("unexpected revision statuses")
	}
	t.Log("History: OK")

	// --- Rollback to v1 ---
	targetRelease, _ := testClient.GetRelease(ns, name, 1)
	testClient.Apply([]byte(targetRelease.Manifest))

	currentV2, _ := testClient.GetCurrentRelease(ns, name)
	testClient.UpdateReleaseStatus(currentV2, k8s.StatusSuperseded)

	prevRev := 1
	releaseV3 := &k8s.Release{
		Name: name, Revision: 3, Status: k8s.StatusDeployed,
		Namespace: ns, Manifest: targetRelease.Manifest, DeployedAt: time.Now(),
		Trigger: k8s.TriggerRollback, PreviousRevision: &prevRev,
		Resources: k8s.ExtractResources(targetRelease.Manifest),
		ResourceCount: len(k8s.ExtractResources(targetRelease.Manifest)),
		Runtime: k8s.CollectRuntime(), Deployer: k8s.CollectDeployer(),
	}
	testClient.SaveRelease(releaseV3)
	t.Log("Rollback: OK")

	// --- Verify 3 revisions ---
	releases, _ = testClient.ListReleases(ns, name)
	if len(releases) != 3 {
		t.Fatalf("expected 3 revisions, got %d", len(releases))
	}
	if releases[2].Trigger != k8s.TriggerRollback {
		t.Fatal("expected v3 trigger=rollback")
	}
	if releases[2].PreviousRevision == nil || *releases[2].PreviousRevision != 1 {
		t.Fatal("expected v3 previousRevision=1")
	}

	// --- Uninstall ---
	currentV3, _ := testClient.GetCurrentRelease(ns, name)
	testClient.Delete([]byte(currentV3.Manifest))
	testClient.DeleteReleases(ns, name)
	t.Log("Uninstall: OK")

	// Verify: no releases left
	releases, _ = testClient.ListReleases(ns, name)
	if len(releases) != 0 {
		t.Fatalf("expected 0 releases after uninstall, got %d", len(releases))
	}

	// Cleanup namespace
	testClient.Delete([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))
}

// ==================== $cluster Pipeline Test ====================

func TestClusterGlobalInPipeline(t *testing.T) {
	code := `
		var version = $cluster.version();
		var nodes = $cluster.nodeCount();
		var hasApps = $cluster.apiAvailable("apps/v1");

		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-cluster-test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "cluster-info" },
				data: {
					version: version,
					nodes: String(nodes),
					hasApps: String(hasApps)
				}
			}]
		})
	`

	dir := t.TempDir()
	tsFile := filepath.Join(dir, "chart.ts")
	os.WriteFile(tsFile, []byte(code), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	jsCode, err := ts.Load(tsFile, false)
	if err != nil {
		t.Fatal(err)
	}

	export, err := ts.Run(jsCode, tsFile, ts.AllPermissions())
	if err != nil {
		t.Fatalf("Run with $cluster failed: %v", err)
	}

	data := export.Components[0]["data"].(map[string]interface{})
	t.Logf("$cluster.version() = %s", data["version"])
	t.Logf("$cluster.nodeCount() = %s", data["nodes"])
	t.Logf("$cluster.apiAvailable('apps/v1') = %s", data["hasApps"])

	if data["version"] == "" {
		t.Fatal("expected non-empty version")
	}
	if data["hasApps"] != "true" {
		t.Fatal("expected apps/v1 to be available")
	}

	// Apply and cleanup
	chart := k8s.PatchAndTransform(export)
	testClient.Apply([]byte(chart.Combined()))

	// Verify ConfigMap was created with cluster data
	if !testClient.ResourceExists("ConfigMap", "c8x-cluster-test", "cluster-info") {
		t.Fatal("ConfigMap cluster-info should exist")
	}

	// Read it back via K8s API
	cms, _ := testClient.ListResources("ConfigMap", "c8x-cluster-test")
	found := false
	for _, cm := range cms {
		meta, _ := cm["metadata"].(map[string]interface{})
		if meta["name"] == "cluster-info" {
			found = true
			cmData, _ := cm["data"].(map[string]interface{})
			if cmData["version"] == "" {
				t.Fatal("ConfigMap version field empty")
			}
		}
	}
	if !found {
		t.Fatal("cluster-info ConfigMap not found via ListResources")
	}

	// Cleanup
	testClient.Delete([]byte(chart.Combined()))
	testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-cluster-test`))
}

// ==================== Release Metadata Verification ====================

func TestReleaseMetadataStored(t *testing.T) {
	ns := "c8x-meta-test"
	testClient.Apply([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))
	defer testClient.Delete([]byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, ns)))

	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: c8x-meta-test
data:
  key: value
---
apiVersion: v1
kind: Service
metadata:
  name: app-svc
  namespace: c8x-meta-test
spec:
  ports:
    - port: 80`

	resources := k8s.ExtractResources(manifest)

	release := &k8s.Release{
		Name: "meta-test", Revision: 1, Status: k8s.StatusDeployed,
		Namespace: ns, Manifest: manifest, DeployedAt: time.Now(),
		Permissions:   &k8s.ReleasePermissions{File: true, Http: false, Cluster: true},
		Resources:     resources,
		ResourceCount: len(resources),
		Duration:      "1.5s",
		Trigger:       k8s.TriggerManual,
		Source:        &k8s.ReleaseSource{File: "chart.ts", Checksum: "sha256:abc123"},
		Runtime:       k8s.CollectRuntime(),
		Deployer:      k8s.CollectDeployer(),
	}

	testClient.SaveRelease(release)

	// Read back and verify all metadata
	stored, err := testClient.GetCurrentRelease(ns, "meta-test")
	if err != nil || stored == nil {
		t.Fatalf("failed to read release: %v", err)
	}

	if stored.Permissions == nil || !stored.Permissions.File || stored.Permissions.Http || !stored.Permissions.Cluster {
		t.Fatal("permissions mismatch")
	}
	if len(stored.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(stored.Resources))
	}
	if stored.ResourceCount != 2 {
		t.Fatal("resourceCount mismatch")
	}
	if stored.Duration != "1.5s" {
		t.Fatal("duration mismatch")
	}
	if stored.Source == nil || stored.Source.File != "chart.ts" {
		t.Fatal("source mismatch")
	}
	if stored.Runtime == nil || stored.Runtime.OS == "" {
		t.Fatal("runtime mismatch")
	}
	if stored.Deployer == nil || stored.Deployer.Hostname == "" {
		t.Fatal("deployer mismatch")
	}

	// Verify ConfigMap exists in cluster with correct labels
	cms, _ := testClient.ListResources("ConfigMap", ns)
	foundReleaseCM := false
	for _, cm := range cms {
		meta, _ := cm["metadata"].(map[string]interface{})
		if meta["name"] == "c8x.release.meta-test.v1" {
			foundReleaseCM = true
			labels, _ := meta["labels"].(map[string]interface{})
			if labels[k8s.LabelManagedBy] != k8s.ManagedByValue {
				t.Fatal("missing managed-by label")
			}
			if labels[k8s.LabelRelease] != "meta-test" {
				t.Fatal("missing release label")
			}
			if labels[k8s.LabelStatus] != k8s.StatusDeployed {
				t.Fatal("wrong status label")
			}
		}
	}
	if !foundReleaseCM {
		t.Fatal("release ConfigMap not found in cluster")
	}
}
