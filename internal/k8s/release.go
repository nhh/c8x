package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusDeployed   = "deployed"
	StatusSuperseded = "superseded"
	StatusFailed     = "failed"

	LabelManagedBy  = "app.kubernetes.io/managed-by"
	LabelRelease    = "c8x.io/release-name"
	LabelRevision   = "c8x.io/revision"
	LabelStatus     = "c8x.io/status"
	ManagedByValue  = "c8x"
)

// Release represents a deployed chart revision stored as a ConfigMap.
type Release struct {
	Name         string            `json:"name"`
	Revision     int               `json:"revision"`
	Status       string            `json:"status"`
	ChartName    string            `json:"chartName"`
	ChartVersion string            `json:"chartVersion"`
	Namespace    string            `json:"namespace"`
	Manifest     string            `json:"manifest"`
	Env          map[string]string `json:"env,omitempty"`
	DeployedAt   time.Time         `json:"deployedAt"`
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
		Data: map[string]string{
			"release": string(data),
		},
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
