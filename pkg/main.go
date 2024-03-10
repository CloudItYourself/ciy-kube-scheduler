package main

import (
	"fmt"
	"os"

	ciySort "github.com/CloudItYourself/ciy-kube-scheduler/pkg/ciy_sort_plugin"

	"k8s.io/component-base/cli"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin("CiySortPlugin", ciySort.New))
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	code := cli.Run(command)
	os.Exit(code)
}
