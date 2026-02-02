/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"     // Set to git tag or version during build
	BuildTime = "unknown" // Set to build timestamp during build
	GitCommit = "unknown" // Set to git commit hash during build
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, build time, and git commit hash of wifimgr.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// printVersion prints version information to stdout
func printVersion() {
	fmt.Printf("wifimgr version %s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}

// GetBuildTime returns the build time string
func GetBuildTime() string {
	return BuildTime
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return GitCommit
}
