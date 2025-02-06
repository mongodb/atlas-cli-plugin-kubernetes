// Copyright 2025 MongoDB Inc
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

package kubernetes

import (
	"errors"
	"log"
	"os"
	"syscall"

	coreConfig "github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli/kubernetes/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli/kubernetes/operator"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/sighandle"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/telemetry"

	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	const use = "kubernetes"

	cmd := &cobra.Command{
		Use:   use,
		Short: "Manage Kubernetes resources.",
		Long:  `This command provides access to Kubernetes features within Atlas.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := coreConfig.LoadAtlasCLIConfig()
			if err != nil {
				log.Fatalf("Failed to load Atlas CLI config: %v", err)
			}
			telemetry.StartTrackingCommand(cmd, args)
			handleSignal()
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			telemetry.FinishTrackingCommand(telemetry.TrackOptions{})
		},
	}

	cmd.AddCommand(config.Builder(), operator.Builder())
	return cmd
}

func handleSignal() {
	sighandle.Notify(func(sig os.Signal) {
		telemetry.FinishTrackingCommand(telemetry.TrackOptions{
			Err:    errors.New(sig.String()),
			Signal: sig.String(),
		})
		os.Exit(1)
	}, os.Interrupt, syscall.SIGTERM)
}
