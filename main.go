package main

import (
	"github.com/kubernetix/k8x/v1/cmd"
	"github.com/kubernetix/k8x/v1/internal/dotenv"
	"runtime"
)

func init() {
	runtime.LockOSThread()
	err := dotenv.Load()

	if err != nil {
		panic(err)
	}
}

func main() {
	cmd.Execute()
}
