// Copyright 2022 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"os"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli/kubernetes"

	"github.com/mongodb-labs/cobra2snooty"
	"github.com/spf13/cobra"
)

func setDisableAutoGenTag(cmd *cobra.Command) {
	cmd.DisableAutoGenTag = true
	for _, cmd := range cmd.Commands() {
		setDisableAutoGenTag(cmd)
	}
}

func main() {
	if err := os.RemoveAll("./docs/command"); err != nil {
		log.Fatal(err)
	}

	const docsPermissions = 0766
	if err := os.MkdirAll("./docs/command", docsPermissions); err != nil {
		log.Fatal(err)
	}

	pluginBuilder := mockAtlasBuilder()

	setDisableAutoGenTag(pluginBuilder)

	if err := cobra2snooty.GenTreeDocs(pluginBuilder, "./docs/command"); err != nil {
		log.Fatal(err)
	}

	filePath := "./docs/command/atlas.txt"

	err := os.Remove(filePath)
	if err != nil {
		log.Fatal(err)
	}

}

// mockAtlasBuilder provides a root command to the kubernetes tasks.
// This ensures that docs generated follow the same structure as in
// AtlasCLI.
func mockAtlasBuilder() *cobra.Command {
	const use = "atlas"

	cmd := &cobra.Command{
		Use: use,
	}

	cmd.AddCommand(kubernetes.Builder())
	return cmd
}
