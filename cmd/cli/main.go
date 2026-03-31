package main

import (
	"fmt"
	"runtime"

	"github.com/lupguo/linkstash/cmd/cli/cmd"
)

var Version = "dev"
var BuildTime = ""
var GitCommit = ""

func main() {
	cmd.RootCmd.Version = fmt.Sprintf("%s (%s)\nBuild: %s\nGo:    %s", Version, GitCommit, BuildTime, runtime.Version())
	cmd.Execute()
}
