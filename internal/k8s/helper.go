package k8s

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"strings"
)

type Chart struct {
	Namespace string
	Content   string
}

func PatchAndTransform(export ChartExport) Chart {
	content := []string{""}

	chart := Chart{}

	for _, component := range export.Components {
		if component == nil {
			continue
		}
		bts, _ := yaml.Marshal(map[string]interface{}(component))
		content = append(content, string(bts))
	}

	chart.Content = strings.Join(content, "---\n")

	if export.Namespace != nil {
		nsyml, _ := yaml.Marshal(map[string]interface{}(export.Namespace))
		chart.Namespace = string(nsyml)
	}

	return chart
}

func kubectlApply(f *os.File) (string, error) {
	cmd := exec.Command("kubectl", "apply", "-f", f.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl apply: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func ApplyChart(chart Chart) error {
	f, err := os.CreateTemp("", "c8x-tmpfile-")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if chart.Namespace != "" {
		if _, err := f.Write([]byte(chart.Namespace)); err != nil {
			return fmt.Errorf("writing namespace to temp file: %w", err)
		}

		output, err := kubectlApply(f)
		if err != nil {
			return err
		}
		fmt.Print(output)

		if err := f.Truncate(0); err != nil {
			return fmt.Errorf("truncating temp file: %w", err)
		}
		if _, err := f.Seek(0, 0); err != nil {
			return fmt.Errorf("seeking temp file: %w", err)
		}
	}

	if _, err := f.Write([]byte(chart.Content)); err != nil {
		return fmt.Errorf("writing chart to temp file: %w", err)
	}

	output, err := kubectlApply(f)
	if err != nil {
		return err
	}
	fmt.Print(output)

	return nil
}

func (chart *Chart) Combined() string {
	arr := make([]string, 2)
	arr[0] = chart.Namespace
	arr[1] = chart.Content
	return strings.Join(arr, "---\n")
}
