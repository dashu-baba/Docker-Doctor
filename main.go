package main

import (
	"github.com/dashu-baba/docker-doctor/cmd"
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