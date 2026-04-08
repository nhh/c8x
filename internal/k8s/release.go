package k8s

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusDeployed   = "deployed"
	StatusSuperseded = "superseded"
	StatusFailed     = "failed"

	TriggerManual   = "manual"
	TriggerCI       = "ci"
	TriggerRollback = "rollback"

	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelRelease   = "c8x.io/release-name"
	LabelRevision  = "c8x.io/revision"
	LabelStatus    = "c8x.io/status"
	ManagedByValue = "c8x"
)

// Release represents a deployed chart revision stored as a ConfigMap.
type Release struct {
	Name         string    `json:"name"`
	Revision     int       `json:"revision"`
	Status       string    `json:"status"`
	ChartName    string    `json:"chartName"`
	ChartVersion string    `json:"chartVersion"`
	Namespace    string    `json:"namespace"`
	Manifest     string    `json:"manifest"`
	DeployedAt   time.Time `json:"deployedAt"`

	// Metadata
	Permissions      *ReleasePermissions `json:"permissions,omitempty"`
	Resources        []string            `json:"resources,omitempty"`
	ResourceCount    int                 `json:"resourceCount"`
	Duration         string              `json:"duration,omitempty"`
	Trigger          string              `json:"trigger"`
	PreviousRevision *int                `json:"previousRevision,omitempty"`
	Source           *ReleaseSource      `json:"source,omitempty"`
	Runtime          *ReleaseRuntime     `json:"runtime,omitempty"`
	Deployer         *ReleaseDeployer    `json:"deployer,omitempty"`
	CI               *ReleaseCI          `json:"ci,omitempty"`
}

type ReleasePermissions struct {
	File    bool `json:"file"`
	Http    bool `json:"http"`
	Cluster bool `json:"cluster"`
}

type ReleaseSource struct {
	File     string `json:"file"`
	Checksum string `json:"checksum"`
}

type ReleaseRuntime struct {
	C8xVersion string `json:"c8xVersion"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

type ReleaseDeployer struct {
	Hostname string `json:"hostname"`
	User     string `json:"user"`
}

type ReleaseCI struct {
	Provider string `json:"provider"`
	RunID    string `json:"runId,omitempty"`
	Actor    string `json:"actor,omitempty"`
	Ref      string `json:"ref,omitempty"`
}

// ExtractResources parses a YAML manifest and returns resource identifiers like "Deployment/app".
func ExtractResources(manifest string) []string {
	var resources []string
	for _, doc := range strings.Split(manifest, "---") {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var kind, name string
		for _, line := range strings.Split(doc, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "kind:") {
				kind = strings.TrimSpace(strings.TrimPrefix(trimmed, "kind:"))
			}
			if strings.HasPrefix(trimmed, "name:") && name == "" {
				name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			}
		}
		if kind != "" && name != "" {
			resources = append(resources, kind+"/"+name)
		}
	}
	return resources
}

// DetectCI detects CI environment from well-known environment variables.
func DetectCI() *ReleaseCI {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return &ReleaseCI{
			Provider: "github",
			RunID:    os.Getenv("GITHUB_RUN_ID"),
			Actor:    os.Getenv("GITHUB_ACTOR"),
			Ref:      os.Getenv("GITHUB_REF"),
		}
	}
	if os.Getenv("GITLAB_CI") == "true" {
		return &ReleaseCI{
			Provider: "gitlab",
			RunID:    os.Getenv("CI_PIPELINE_ID"),
			Actor:    os.Getenv("GITLAB_USER_LOGIN"),
			Ref:      os.Getenv("CI_COMMIT_REF_NAME"),
		}
	}
	if os.Getenv("JENKINS_URL") != "" {
		return &ReleaseCI{
			Provider: "jenkins",
			RunID:    os.Getenv("BUILD_NUMBER"),
			Actor:    os.Getenv("BUILD_USER"),
			Ref:      os.Getenv("GIT_BRANCH"),
		}
	}
	return nil
}

// CollectDeployer gathers info about the machine performing the deploy.
func CollectDeployer() *ReleaseDeployer {
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows
	}
	return &ReleaseDeployer{Hostname: hostname, User: user}
}

// CollectRuntime gathers c8x runtime info.
func CollectRuntime() *ReleaseRuntime {
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	return &ReleaseRuntime{C8xVersion: version, OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// CollectSource gathers info about the chart source file.
func CollectSource(filePath string) *ReleaseSource {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &ReleaseSource{File: filePath}
	}
	hash := sha256.Sum256(data)
	return &ReleaseSource{File: filePath, Checksum: fmt.Sprintf("sha256:%x", hash)}
}

func releaseConfigMapName(name string, revision int) string {
	return fmt.Sprintf("c8x.release.%s.v%d", name, revision)
}

func releaseLabels(r *Release) map[string]string {
	return map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelRelease:   r.Name,
		LabelRevision:  strconv.Itoa(r.Revision),
		LabelStatus:    r.Status,
	}
}

// SaveRelease stores a release as a ConfigMap in the cluster.
func (c *Client) SaveRelease(r *Release) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshaling release: %w", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      releaseConfigMapName(r.Name, r.Revision),
			Namespace: r.Namespace,
			Labels:    releaseLabels(r),
		},
		Data: map[string]string{"release": string(data)},
	}

	_, err = c.clientset.CoreV1().ConfigMaps(r.Namespace).Create(
		context.TODO(), cm, metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("saving release %s v%d: %w", r.Name, r.Revision, err)
	}
	return nil
}

// UpdateReleaseStatus updates the status label and data of an existing release.
func (c *Client) UpdateReleaseStatus(r *Release, status string) error {
	cmName := releaseConfigMapName(r.Name, r.Revision)
	cm, err := c.clientset.CoreV1().ConfigMaps(r.Namespace).Get(
		context.TODO(), cmName, metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("getting release %s v%d: %w", r.Name, r.Revision, err)
	}

	r.Status = status
	cm.Labels[LabelStatus] = status

	data, _ := json.Marshal(r)
	cm.Data["release"] = string(data)

	_, err = c.clientset.CoreV1().ConfigMaps(r.Namespace).Update(
		context.TODO(), cm, metav1.UpdateOptions{},
	)
	return err
}

// GetCurrentRelease returns the currently deployed release, or nil if none exists.
func (c *Client) GetCurrentRelease(namespace, name string) (*Release, error) {
	cms, err := c.clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s,%s=%s",
				LabelManagedBy, ManagedByValue,
				LabelRelease, name,
				LabelStatus, StatusDeployed,
			),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}
	if len(cms.Items) == 0 {
		return nil, nil
	}
	return parseRelease(&cms.Items[0])
}

// GetRelease returns a specific revision of a release.
func (c *Client) GetRelease(namespace, name string, revision int) (*Release, error) {
	cmName := releaseConfigMapName(name, revision)
	cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(
		context.TODO(), cmName, metav1.GetOptions{},
	)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("release %s v%d not found", name, revision)
		}
		return nil, err
	}
	return parseRelease(cm)
}

// ListReleases returns all revisions of a release, sorted by revision number.
func (c *Client) ListReleases(namespace, name string) ([]*Release, error) {
	cms, err := c.clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
				LabelManagedBy, ManagedByValue,
				LabelRelease, name,
			),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}

	releases := make([]*Release, 0, len(cms.Items))
	for i := range cms.Items {
		r, err := parseRelease(&cms.Items[i])
		if err != nil {
			continue
		}
		releases = append(releases, r)
	}

	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Revision < releases[j].Revision
	})
	return releases, nil
}

// DeleteReleases deletes all release ConfigMaps for a given name.
func (c *Client) DeleteReleases(namespace, name string) error {
	return c.clientset.CoreV1().ConfigMaps(namespace).DeleteCollection(
		context.TODO(),
		metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
				LabelManagedBy, ManagedByValue,
				LabelRelease, name,
			),
		},
	)
}

// DeleteOldRevisions keeps only the latest N revisions, deleting the rest.
func (c *Client) DeleteOldRevisions(namespace, name string, keepMax int) error {
	releases, err := c.ListReleases(namespace, name)
	if err != nil {
		return err
	}
	if len(releases) <= keepMax {
		return nil
	}
	toDelete := releases[:len(releases)-keepMax]
	for _, r := range toDelete {
		cmName := releaseConfigMapName(r.Name, r.Revision)
		err := c.clientset.CoreV1().ConfigMaps(namespace).Delete(
			context.TODO(), cmName, metav1.DeleteOptions{},
		)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting old revision %s v%d: %w", name, r.Revision, err)
		}
	}
	return nil
}

func parseRelease(cm *corev1.ConfigMap) (*Release, error) {
	data, ok := cm.Data["release"]
	if !ok {
		return nil, fmt.Errorf("configmap %s has no release data", cm.Name)
	}
	var r Release
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return nil, fmt.Errorf("parsing release from %s: %w", cm.Name, err)
	}
	return &r, nil
}
