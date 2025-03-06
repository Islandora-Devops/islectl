/*
Copyright © 2025 Islandora Foundation
*/
package main

import "github.com/islandora-devops/islectl/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
