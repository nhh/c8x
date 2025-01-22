package k8s

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Chart struct {
	Namespace string
	Content   string
}

func HasValidNamespace(namespace interface{}) bool {
	if namespace == nil {
		return false
	}

	if namespace == "" {
		return false
	}

	return true
}

func PatchAndTransform(export map[string]interface{}) Chart {
	content := []string{""}

	chart := Chart{}

	for _, component := range export["components"].([]interface{}) {
		if component == nil {
			continue
		}
		bts, _ := yaml.Marshal(component)
		content = append(content, string(bts))
	}

	chart.Content = strings.Join(content, "---\n")

	namespace := export["namespace"]

	if HasValidNamespace(namespace) {
		nsyml, _ := yaml.Marshal(namespace)
		chart.Namespace = string(nsyml)
	}

	return chart
}

// Todo add error handling
func ApplyChart(chart Chart) {
	// create and open a temporary file
	f, err := os.CreateTemp("", "k8x-tmpfile-") // in Go version older than 1.17 you can use ioutil.TempFile
	if err != nil {
		log.Fatal(err)
	}
	// close and remove the temporary file at the end of the program
	defer f.Close()
	defer os.Remove(f.Name())

	if chart.Namespace == "" {
		if _, err := f.Write([]byte(chart.Namespace)); err != nil {
			log.Fatal(err)
		}

		//fileOutput, _ := os.ReadFile(f.Name())
		//fmt.Println(string(fileOutput))

		grepCmd := exec.Command("kubectl", "apply", "-f", f.Name())

		output, _ := grepCmd.Output()
		fmt.Print(string(output))

		if strings.Contains(string(output), "created") {
			time.Sleep(1 * time.Second)
		}

		// Reset file
		err = f.Truncate(0)
		if err != nil {
			return
		}

		_, err = f.Seek(0, 0)
	}

	// Write chart
	if _, err := f.Write([]byte(chart.Content)); err != nil {
		log.Fatal(err)
	}

	//fileOutput, _ = os.ReadFile(f.Name())
	//fmt.Println(string(fileOutput))

	grepCmd := exec.Command("kubectl", "apply", "-f", f.Name())

	output, _ := grepCmd.Output()
	fmt.Println(string(output))
}

func (chart *Chart) Combined() string {
	arr := make([]string, 2)
	arr[0] = chart.Namespace
	arr[1] = chart.Content
	return strings.Join(arr, "---\n")
}
