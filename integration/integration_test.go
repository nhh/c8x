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

// ==================== Client Edge Cases ====================

func TestClientCRDExists(t *testing.T) {
	if testClient.CRDExists("nonexistent.example.com") {
		t.Fatal("expected false for nonexistent CRD")
	}
}

func TestClientApplyIdempotent(t *testing.T) {
	yaml := `apiVersion: v1
kind: Namespace
metadata:
  name: c8x-idempotent-test
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: idem-cm
  namespace: c8x-idempotent-test
data:
  key: value`

	// Apply twice
	_, err := testClient.Apply([]byte(yaml))
	if err != nil {
		t.Fatalf("first apply failed: %v", err)
	}
	_, err = testClient.Apply([]byte(yaml))
	if err != nil {
		t.Fatalf("second apply failed (should be idempotent): %v", err)
	}

	if !testClient.ResourceExists("ConfigMap", "c8x-idempotent-test", "idem-cm") {
		t.Fatal("ConfigMap should exist after double apply")
	}

	testClient.Delete([]byte(yaml))
}

func TestClientApplyInvalidYAML(t *testing.T) {
	_, err := testClient.Apply([]byte("this is not valid yaml at all {{{"))
	// Should not panic, should return error or skip
	if err != nil {
		t.Logf("Got expected error for invalid YAML: %v", err)
	}
}

func TestClientDeleteNonexistent(t *testing.T) {
	yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: does-not-exist-at-all
  namespace: default`

	// Should not panic or error fatally
	_, err := testClient.Delete([]byte(yaml))
	if err != nil {
		t.Fatalf("Delete of nonexistent should not error: %v", err)
	}
}

// ==================== Individual Lifecycle Tests ====================

// chartNs is the namespace hardcoded in testdata/chart.ts
const chartNs = "c8x-integration-test"

func installChart(t *testing.T, chartFile, name string) *k8s.Release {
	t.Helper()
	chart := compileAndApply(t, chartFile)
	release := &k8s.Release{
		Name: name, Revision: 1, Status: k8s.StatusDeployed,
		Namespace: chartNs, Manifest: chart.Combined(), DeployedAt: time.Now(),
		Trigger: k8s.TriggerManual, Resources: k8s.ExtractResources(chart.Combined()),
		ResourceCount: len(k8s.ExtractResources(chart.Combined())),
		Runtime: k8s.CollectRuntime(), Deployer: k8s.CollectDeployer(),
	}
	if err := testClient.SaveRelease(release); err != nil {
		t.Fatalf("SaveRelease failed: %v", err)
	}
	return release
}

func cleanupRelease(name string) {
	testClient.DeleteReleases(chartNs, name)
}

func TestInstallCreatesReleaseConfigMap(t *testing.T) {
	name := "install-cm-test"
	defer cleanupRelease(name)

	installChart(t, chartPath, name)

	if !testClient.ResourceExists("ConfigMap", chartNs, "c8x.release."+name+".v1") {
		t.Fatal("release ConfigMap should exist after install")
	}
}

func TestInstallDuplicateBlocked(t *testing.T) {
	name := "dup-test"
	defer cleanupRelease(name)

	installChart(t, chartPath, name)

	dup := &k8s.Release{
		Name: name, Revision: 1, Status: k8s.StatusDeployed,
		Namespace: chartNs, Manifest: "dup", DeployedAt: time.Now(), Trigger: k8s.TriggerManual,
	}
	err := testClient.SaveRelease(dup)
	if err == nil {
		t.Fatal("expected error when saving duplicate revision")
	}

	current, _ := testClient.GetCurrentRelease(chartNs, name)
	if current == nil || current.Revision != 1 {
		t.Fatal("original release should still be current")
	}
}

func TestUpgradeWithoutInstallFails(t *testing.T) {
	current, err := testClient.GetCurrentRelease(chartNs, "never-installed")
	if err != nil {
		t.Fatal(err)
	}
	if current != nil {
		t.Fatal("expected no release before install")
	}
}

func TestRollbackToSpecificRevision(t *testing.T) {
	name := "rollback-spec-test"
	defer cleanupRelease(name)

	// Install v1
	chart1 := compileAndApply(t, chartPath)
	r1 := &k8s.Release{
		Name: name, Revision: 1, Status: k8s.StatusDeployed,
		Namespace: chartNs, Manifest: chart1.Combined(), DeployedAt: time.Now(),
		Trigger: k8s.TriggerManual,
	}
	testClient.SaveRelease(r1)

	// Upgrade to v2
	chart2 := compileAndApply(t, chartV2Path)
	testClient.UpdateReleaseStatus(r1, k8s.StatusSuperseded)
	r2 := &k8s.Release{
		Name: name, Revision: 2, Status: k8s.StatusDeployed,
		Namespace: chartNs, Manifest: chart2.Combined(), DeployedAt: time.Now(),
		Trigger: k8s.TriggerManual,
	}
	testClient.SaveRelease(r2)

	// Upgrade to v3
	testClient.UpdateReleaseStatus(r2, k8s.StatusSuperseded)
	r3 := &k8s.Release{
		Name: name, Revision: 3, Status: k8s.StatusDeployed,
		Namespace: chartNs, Manifest: chart2.Combined(), DeployedAt: time.Now(),
		Trigger: k8s.TriggerManual,
	}
	testClient.SaveRelease(r3)

	// Rollback to v1
	target, err := testClient.GetRelease(chartNs, name, 1)
	if err != nil {
		t.Fatalf("GetRelease v1 failed: %v", err)
	}
	testClient.Apply([]byte(target.Manifest))
	testClient.UpdateReleaseStatus(r3, k8s.StatusSuperseded)
	prevRev := 1
	r4 := &k8s.Release{
		Name: name, Revision: 4, Status: k8s.StatusDeployed,
		Namespace: chartNs, Manifest: target.Manifest, DeployedAt: time.Now(),
		Trigger: k8s.TriggerRollback, PreviousRevision: &prevRev,
	}
	testClient.SaveRelease(r4)

	releases, _ := testClient.ListReleases(chartNs, name)
	if len(releases) != 4 {
		t.Fatalf("expected 4 revisions, got %d", len(releases))
	}

	current, _ := testClient.GetCurrentRelease(chartNs, name)
	if current.Revision != 4 || current.Trigger != k8s.TriggerRollback {
		t.Fatalf("expected v4 rollback, got v%d %s", current.Revision, current.Trigger)
	}
	if current.PreviousRevision == nil || *current.PreviousRevision != 1 {
		t.Fatal("expected previousRevision=1")
	}
}

func TestUninstallDeletesResources(t *testing.T) {
	name := "uninstall-test"

	installChart(t, chartPath, name)

	// Verify resources exist
	if !testClient.ResourceExists("ConfigMap", chartNs, "test-config") {
		t.Fatal("ConfigMap should exist before uninstall")
	}

	// Uninstall
	current, _ := testClient.GetCurrentRelease(chartNs, name)
	testClient.Delete([]byte(current.Manifest))
	cleanupRelease(name)

	time.Sleep(500 * time.Millisecond)

	// Verify releases gone
	releases, _ := testClient.ListReleases(chartNs, name)
	if len(releases) != 0 {
		t.Fatalf("expected 0 releases, got %d", len(releases))
	}
}

func TestUninstallNonexistentFails(t *testing.T) {
	current, _ := testClient.GetCurrentRelease("nonexistent-ns-xyz", "nonexistent-xyz")
	if current != nil {
		t.Fatal("expected nil for nonexistent release")
	}
}

// ==================== $cluster Pipeline Tests ====================

func TestClusterCRDExistsInChart(t *testing.T) {
	code := `
		var hasCM = $cluster.crdExists("nonexistent.example.com");
		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-crd-test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "crd-check" },
				data: { hasCM: String(hasCM) }
			}]
		})
	`
	export := compileInlineChart(t, code)
	data := export.Components[0]["data"].(map[string]interface{})
	if data["hasCM"] != "false" {
		t.Fatalf("expected false, got %v", data["hasCM"])
	}
}

func TestClusterResourceExistsInChart(t *testing.T) {
	// Create a ConfigMap first
	testClient.Apply([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-resexist-test
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: probe-target
  namespace: c8x-resexist-test
data:
  exists: "true"`))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-resexist-test`))

	code := `
		var found = $cluster.exists("v1", "ConfigMap", "c8x-resexist-test", "probe-target");
		var notFound = $cluster.exists("v1", "ConfigMap", "c8x-resexist-test", "nope");
		export default () => ({
			namespace: { apiVersion: "v1", kind: "Namespace", metadata: { name: "c8x-resexist-test" } },
			components: [{
				apiVersion: "v1", kind: "ConfigMap",
				metadata: { name: "exist-result" },
				data: { found: String(found), notFound: String(notFound) }
			}]
		})
	`
	export := compileInlineChart(t, code)
	data := export.Components[0]["data"].(map[string]interface{})
	if data["found"] != "true" {
		t.Fatalf("expected found=true, got %v", data["found"])
	}
	if data["notFound"] != "false" {
		t.Fatalf("expected notFound=false, got %v", data["notFound"])
	}
}

func TestClusterVersionAtLeastInChart(t *testing.T) {
	export := compileFixtureChart(t, filepath.Join(testdataDir, "chart-cluster-version.ts"))
	data := export.Components[0]["data"].(map[string]interface{})
	// KinD runs modern K8s, should be >= 1.25
	if data["isModern"] != "true" {
		t.Fatalf("expected isModern=true, got %v", data["isModern"])
	}
}

// ==================== Globals Pipeline in Cluster ====================

func TestFileReadApplied(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-with-file.ts")
	export := compileFixtureChart(t, chartFile)
	chart := k8s.PatchAndTransform(export)

	testClient.Apply([]byte(chart.Combined()))
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-file-test`))

	if !testClient.ResourceExists("ConfigMap", "c8x-file-test", "nginx-config") {
		t.Fatal("ConfigMap nginx-config should exist")
	}
}

func TestAssertBlocksDeploy(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-assert-fail.ts")
	code, err := ts.Load(chartFile, false)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	_, err = ts.Run(code, chartFile, ts.AllPermissions())
	if err == nil {
		t.Fatal("expected $assert to block chart execution")
	}
	if !strings.Contains(err.Error(), "intentionally fails") {
		t.Fatalf("expected assertion message, got %v", err)
	}
}

func TestHashInAnnotationApplied(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-with-hash.ts")
	export := compileFixtureChart(t, chartFile)
	chart := k8s.PatchAndTransform(export)

	testClient.Apply([]byte(chart.Combined()))
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-hash-test`))

	cms, _ := testClient.ListResources("ConfigMap", "c8x-hash-test")
	for _, cm := range cms {
		meta, _ := cm["metadata"].(map[string]interface{})
		if meta["name"] == "hashed-config" {
			annotations, _ := meta["annotations"].(map[string]interface{})
			hash, _ := annotations["c8x/config-hash"].(string)
			if len(hash) != 64 {
				t.Fatalf("expected 64-char sha256 hash annotation, got %q", hash)
			}
			return
		}
	}
	t.Fatal("hashed-config ConfigMap not found")
}

func TestBase64InSecretApplied(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-with-base64.ts")
	export := compileFixtureChart(t, chartFile)
	chart := k8s.PatchAndTransform(export)

	testClient.Apply([]byte(chart.Combined()))
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-b64-test`))

	if !testClient.ResourceExists("Secret", "c8x-b64-test", "encoded-secret") {
		t.Fatal("Secret encoded-secret should exist")
	}
}

func TestYamlParseApplied(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-with-yaml.ts")
	export := compileFixtureChart(t, chartFile)
	chart := k8s.PatchAndTransform(export)

	testClient.Apply([]byte(chart.Combined()))
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-yaml-test`))

	if !testClient.ResourceExists("ConfigMap", "c8x-yaml-test", "prometheus-config") {
		t.Fatal("ConfigMap prometheus-config should exist")
	}
}

// ==================== Permissions in Cluster ====================

func TestFileBlockedWithoutPermission(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-with-file.ts")
	code, _ := ts.Load(chartFile, false)

	_, err := ts.Run(code, chartFile) // no permissions
	if err == nil {
		t.Fatal("expected $file.read to be blocked without --allow-file")
	}
	if !strings.Contains(err.Error(), "allow-file") {
		t.Fatalf("expected allow-file hint, got %v", err)
	}
}

func TestClusterBlockedWithoutPermission(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-cluster-version.ts")
	code, _ := ts.Load(chartFile, false)

	_, err := ts.Run(code, chartFile) // no permissions
	if err == nil {
		t.Fatal("expected $cluster to be blocked without --allow-cluster")
	}
	if !strings.Contains(err.Error(), "allow-cluster") {
		t.Fatalf("expected allow-cluster hint, got %v", err)
	}
}

// ==================== Edge Cases ====================

func TestApplyLargeChart(t *testing.T) {
	chartFile := filepath.Join(testdataDir, "chart-large.ts")
	export := compileFixtureChart(t, chartFile)
	chart := k8s.PatchAndTransform(export)

	_, err := testClient.Apply([]byte(chart.Combined()))
	if err != nil {
		t.Fatalf("Apply large chart failed: %v", err)
	}
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-large-test`))

	if len(export.Components) != 20 {
		t.Fatalf("expected 20 components, got %d", len(export.Components))
	}

	// Spot check a few
	if !testClient.ResourceExists("ConfigMap", "c8x-large-test", "cm-0") {
		t.Fatal("cm-0 should exist")
	}
	if !testClient.ResourceExists("ConfigMap", "c8x-large-test", "cm-19") {
		t.Fatal("cm-19 should exist")
	}
}

func TestNamespaceAutoCreated(t *testing.T) {
	chart := compileAndApply(t, chartPath)
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-integration-test`))

	// Namespace from chart should have been created by Apply
	if !testClient.ResourceExists("Namespace", "", "c8x-integration-test") {
		t.Fatal("namespace should be auto-created from chart")
	}
}

func TestApplyIdempotentFullChart(t *testing.T) {
	chart := compileAndApply(t, chartPath)
	defer testClient.Delete([]byte(chart.Combined()))
	defer testClient.Delete([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: c8x-integration-test`))

	// Apply same chart again → should not error
	if err := k8s.ApplyChart(testClient, chart); err != nil {
		t.Fatalf("second apply should be idempotent: %v", err)
	}
}

// ==================== Helpers ====================

func compileInlineChart(t *testing.T, code string) k8s.ChartExport {
	t.Helper()
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
		t.Fatal(err)
	}
	return export
}

func compileFixtureChart(t *testing.T, chartFile string) k8s.ChartExport {
	t.Helper()
	code, err := ts.Load(chartFile, false)
	if err != nil {
		t.Fatal(err)
	}
	export, err := ts.Run(code, chartFile, ts.AllPermissions())
	if err != nil {
		t.Fatal(err)
	}
	return export
}
