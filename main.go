package main

import (
	"github.com/example/docker-doctor/cmd"
)

var (
	version   = "dev"
	gitCommit = ""
	buildTime = ""
)

func main() {
	cmd.SetVersion(version, gitCommit, buildTime)
	cmd.Execute()
}