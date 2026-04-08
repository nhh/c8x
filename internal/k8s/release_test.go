package k8s

import (
	"encoding/json"
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
	r := &Release{
		Name:         "wordpress",
		Revision:     1,
		Status:       StatusDeployed,
		ChartName:    "@c8x/wordpress",
		ChartVersion: "0.0.1",
		Namespace:    "wordpress",
		Manifest:     "apiVersion: v1\nkind: Service\n",
		Env:          map[string]string{"WP_DOMAIN": "blog.example.com"},
		DeployedAt:   time.Date(2026, 4, 8, 22, 0, 0, 0, time.UTC),
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
	if parsed.Revision != 1 {
		t.Fatalf("expected 1, got %d", parsed.Revision)
	}
	if parsed.Status != StatusDeployed {
		t.Fatalf("expected deployed, got %s", parsed.Status)
	}
	if parsed.ChartName != "@c8x/wordpress" {
		t.Fatalf("expected @c8x/wordpress, got %s", parsed.ChartName)
	}
	if parsed.Namespace != "wordpress" {
		t.Fatalf("expected wordpress, got %s", parsed.Namespace)
	}
	if parsed.Manifest != "apiVersion: v1\nkind: Service\n" {
		t.Fatalf("manifest mismatch")
	}
	if parsed.Env["WP_DOMAIN"] != "blog.example.com" {
		t.Fatalf("env mismatch")
	}
	if !parsed.DeployedAt.Equal(r.DeployedAt) {
		t.Fatalf("time mismatch")
	}
}

func TestReleaseSerializationEmptyEnv(t *testing.T) {
	r := &Release{
		Name:     "minimal",
		Revision: 1,
		Status:   StatusDeployed,
	}

	data, _ := json.Marshal(r)
	var parsed Release
	json.Unmarshal(data, &parsed)

	if parsed.Env != nil {
		t.Fatalf("expected nil env, got %v", parsed.Env)
	}
}

func TestReleaseSerializationLargeManifest(t *testing.T) {
	manifest := ""
	for i := 0; i < 100; i++ {
		manifest += "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n---\n"
	}

	r := &Release{
		Name:     "large",
		Revision: 1,
		Status:   StatusDeployed,
		Manifest: manifest,
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Release
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Manifest != manifest {
		t.Fatal("manifest roundtrip failed")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusDeployed != "deployed" {
		t.Fatal("wrong constant")
	}
	if StatusSuperseded != "superseded" {
		t.Fatal("wrong constant")
	}
	if StatusFailed != "failed" {
		t.Fatal("wrong constant")
	}
}
