package main

import (
	"github.com/kubernetix/c8x/cmd"
	"github.com/kubernetix/c8x/internal/dotenv"
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
