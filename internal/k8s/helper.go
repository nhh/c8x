package k8s

import (
	"fmt"
	"gopkg.in/yaml.v3"
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

func ApplyChart(client *Client, chart Chart) error {
	if chart.Namespace != "" {
		output, err := client.Apply([]byte(chart.Namespace))
		if err != nil {
			return fmt.Errorf("applying namespace: %w", err)
		}
		if output != "" {
			fmt.Println(output)
		}
	}

	output, err := client.Apply([]byte(chart.Content))
	if err != nil {
		return fmt.Errorf("applying chart: %w", err)
	}
	if output != "" {
		fmt.Println(output)
	}

	return nil
}

func (chart *Chart) Combined() string {
	arr := make([]string, 2)
	arr[0] = chart.Namespace
	arr[1] = chart.Content
	return strings.Join(arr, "---\n")
}
