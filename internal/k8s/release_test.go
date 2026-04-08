package k8s

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReleaseConfigMapName(t *testing.T) {
	if releaseConfigMapName("wordpress", 1) != "c8x.release.wordpress.v1" {
		t.Fatal("unexpected name")
	}
	if releaseConfigMapName("my-app", 42) != "c8x.release.my-app.v42" {
		t.Fatal("unexpected name")
	}
}

func TestReleaseLabels(t *testing.T) {
	r := &Release{Name: "wordpress", Revision: 3, Status: StatusDeployed}
	labels := releaseLabels(r)

	if labels[LabelManagedBy] != ManagedByValue {
		t.Fatalf("expected %s, got %s", ManagedByValue, labels[LabelManagedBy])
	}
	if labels[LabelRelease] != "wordpress" {
		t.Fatalf("expected wordpress, got %s", labels[LabelRelease])
	}
	if labels[LabelRevision] != "3" {
		t.Fatalf("expected 3, got %s", labels[LabelRevision])
	}
	if labels[LabelStatus] != StatusDeployed {
		t.Fatalf("expected deployed, got %s", labels[LabelStatus])
	}
}

func TestReleaseSerialization(t *testing.T) {
	prevRev := 2
	r := &Release{
		Name:         "wordpress",
		Revision:     3,
		Status:       StatusDeployed,
		ChartName:    "@c8x/wordpress",
		ChartVersion: "0.0.1",
		Namespace:    "wordpress",
		Manifest:     "apiVersion: v1\nkind: Service\n",
		DeployedAt:   time.Date(2026, 4, 8, 22, 0, 0, 0, time.UTC),
		Permissions:  &ReleasePermissions{File: true, Http: false, Cluster: true},
		Resources:    []string{"Service/wordpress", "Deployment/wordpress"},
		ResourceCount: 2,
		Duration:     "3.2s",
		Trigger:      TriggerManual,
		PreviousRevision: &prevRev,
		Source:       &ReleaseSource{File: "chart.ts", Checksum: "sha256:abc"},
		Runtime:      &ReleaseRuntime{C8xVersion: "0.0.20", OS: "darwin", Arch: "arm64"},
		Deployer:     &ReleaseDeployer{Hostname: "mbp.local", User: "niklas"},
		CI:           nil,
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Release
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Name != "wordpress" {
		t.Fatalf("expected wordpress, got %s", parsed.Name)
	}
	if parsed.Revision != 3 {
		t.Fatalf("expected 3, got %d", parsed.Revision)
	}
	if !parsed.Permissions.File || parsed.Permissions.Http || !parsed.Permissions.Cluster {
		t.Fatal("permissions mismatch")
	}
	if len(parsed.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(parsed.Resources))
	}
	if parsed.ResourceCount != 2 {
		t.Fatalf("expected count 2, got %d", parsed.ResourceCount)
	}
	if parsed.Duration != "3.2s" {
		t.Fatalf("expected 3.2s, got %s", parsed.Duration)
	}
	if parsed.Trigger != TriggerManual {
		t.Fatalf("expected manual, got %s", parsed.Trigger)
	}
	if parsed.PreviousRevision == nil || *parsed.PreviousRevision != 2 {
		t.Fatal("previousRevision mismatch")
	}
	if parsed.Source.File != "chart.ts" || parsed.Source.Checksum != "sha256:abc" {
		t.Fatal("source mismatch")
	}
	if parsed.Runtime.C8xVersion != "0.0.20" || parsed.Runtime.OS != "darwin" {
		t.Fatal("runtime mismatch")
	}
	if parsed.Deployer.Hostname != "mbp.local" || parsed.Deployer.User != "niklas" {
		t.Fatal("deployer mismatch")
	}
	if parsed.CI != nil {
		t.Fatal("expected nil CI")
	}
}

func TestReleaseSerializationWithCI(t *testing.T) {
	r := &Release{
		Name:     "app",
		Revision: 1,
		Status:   StatusDeployed,
		CI:       &ReleaseCI{Provider: "github", RunID: "123", Actor: "nhh", Ref: "refs/heads/main"},
	}

	data, _ := json.Marshal(r)
	var parsed Release
	json.Unmarshal(data, &parsed)

	if parsed.CI == nil {
		t.Fatal("expected CI")
	}
	if parsed.CI.Provider != "github" || parsed.CI.RunID != "123" || parsed.CI.Actor != "nhh" {
		t.Fatal("CI mismatch")
	}
}

func TestReleaseSerializationMinimal(t *testing.T) {
	r := &Release{Name: "minimal", Revision: 1, Status: StatusDeployed}

	data, _ := json.Marshal(r)
	var parsed Release
	json.Unmarshal(data, &parsed)

	if parsed.Permissions != nil {
		t.Fatal("expected nil permissions")
	}
	if parsed.Source != nil {
		t.Fatal("expected nil source")
	}
	if parsed.CI != nil {
		t.Fatal("expected nil CI")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusDeployed != "deployed" || StatusSuperseded != "superseded" || StatusFailed != "failed" {
		t.Fatal("wrong constant")
	}
}

func TestTriggerConstants(t *testing.T) {
	if TriggerManual != "manual" || TriggerCI != "ci" || TriggerRollback != "rollback" {
		t.Fatal("wrong constant")
	}
}

// --- ExtractResources ---

func TestExtractResourcesSimple(t *testing.T) {
	manifest := `kind: Service
metadata:
  name: app
---
kind: Deployment
metadata:
  name: app`

	resources := ExtractResources(manifest)
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0] != "Service/app" {
		t.Fatalf("expected Service/app, got %s", resources[0])
	}
	if resources[1] != "Deployment/app" {
		t.Fatalf("expected Deployment/app, got %s", resources[1])
	}
}

func TestExtractResourcesEmpty(t *testing.T) {
	resources := ExtractResources("")
	if len(resources) != 0 {
		t.Fatalf("expected 0, got %d", len(resources))
	}
}

func TestExtractResourcesSkipsIncomplete(t *testing.T) {
	manifest := `kind: Service
---
metadata:
  name: orphan`

	resources := ExtractResources(manifest)
	if len(resources) != 0 {
		t.Fatalf("expected 0 (no complete resources), got %d: %v", len(resources), resources)
	}
}

func TestExtractResourcesMultiple(t *testing.T) {
	manifest := `kind: Namespace
metadata:
  name: wp
---
kind: Secret
metadata:
  name: db-creds
---
kind: Deployment
metadata:
  name: wordpress
---
kind: Service
metadata:
  name: wordpress
---
kind: Ingress
metadata:
  name: wordpress`

	resources := ExtractResources(manifest)
	if len(resources) != 5 {
		t.Fatalf("expected 5, got %d", len(resources))
	}
}

// --- DetectCI ---

func TestDetectCIGitHub(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_RUN_ID", "42")
	t.Setenv("GITHUB_ACTOR", "nhh")
	t.Setenv("GITHUB_REF", "refs/heads/main")

	ci := DetectCI()
	if ci == nil {
		t.Fatal("expected CI detected")
	}
	if ci.Provider != "github" || ci.RunID != "42" || ci.Actor != "nhh" {
		t.Fatalf("unexpected: %+v", ci)
	}
}

func TestDetectCIGitLab(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_PIPELINE_ID", "99")
	t.Setenv("GITLAB_USER_LOGIN", "dev")
	t.Setenv("CI_COMMIT_REF_NAME", "main")

	ci := DetectCI()
	if ci == nil {
		t.Fatal("expected CI detected")
	}
	if ci.Provider != "gitlab" || ci.RunID != "99" {
		t.Fatalf("unexpected: %+v", ci)
	}
}

func TestDetectCIJenkins(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "http://jenkins.local")
	t.Setenv("BUILD_NUMBER", "55")

	ci := DetectCI()
	if ci == nil {
		t.Fatal("expected CI detected")
	}
	if ci.Provider != "jenkins" || ci.RunID != "55" {
		t.Fatalf("unexpected: %+v", ci)
	}
}

func TestDetectCINone(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "")

	ci := DetectCI()
	if ci != nil {
		t.Fatalf("expected nil, got %+v", ci)
	}
}

// --- CollectDeployer ---

func TestCollectDeployer(t *testing.T) {
	d := CollectDeployer()
	if d == nil {
		t.Fatal("expected non-nil")
	}
	if d.Hostname == "" {
		t.Fatal("expected hostname")
	}
	// User may be empty in some CI environments, that's ok
}

// --- CollectRuntime ---

func TestCollectRuntime(t *testing.T) {
	rt := CollectRuntime()
	if rt == nil {
		t.Fatal("expected non-nil")
	}
	if rt.OS != runtime.GOOS {
		t.Fatalf("expected %s, got %s", runtime.GOOS, rt.OS)
	}
	if rt.Arch != runtime.GOARCH {
		t.Fatalf("expected %s, got %s", runtime.GOARCH, rt.Arch)
	}
	if rt.C8xVersion == "" {
		t.Fatal("expected non-empty version")
	}
}

// --- CollectSource ---

func TestCollectSource(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/chart.ts"
	os.WriteFile(f, []byte("export default () => ({})"), 0644)

	s := CollectSource(f)
	if s.File != f {
		t.Fatalf("expected %s, got %s", f, s.File)
	}
	if !strings.HasPrefix(s.Checksum, "sha256:") {
		t.Fatalf("expected sha256 prefix, got %s", s.Checksum)
	}
	if len(s.Checksum) != 71 { // "sha256:" + 64 hex chars
		t.Fatalf("expected 71 char checksum, got %d", len(s.Checksum))
	}
}

func TestCollectSourceMissing(t *testing.T) {
	s := CollectSource("/nonexistent/chart.ts")
	if s.File != "/nonexistent/chart.ts" {
		t.Fatal("expected file path even for missing file")
	}
	if s.Checksum != "" {
		t.Fatal("expected empty checksum for missing file")
	}
}

func TestCollectSourceDeterministic(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/chart.ts"
	os.WriteFile(f, []byte("same content"), 0644)

	s1 := CollectSource(f)
	s2 := CollectSource(f)
	if s1.Checksum != s2.Checksum {
		t.Fatal("same file should produce same checksum")
	}
}
