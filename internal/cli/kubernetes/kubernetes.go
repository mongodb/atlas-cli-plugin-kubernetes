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
	"fmt"

	coreConfig "github.com/mongodb/atlas-cli-core/config"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli/kubernetes/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli/kubernetes/operator"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/flag"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/log"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/telemetry"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/usage"

	"github.com/spf13/cobra"
)

func Builder() *cobra.Command {
	const use = "kubernetes"
	var (
		profile    string
		debugLevel bool
	)

	cmd := &cobra.Command{
		Use:   use,
		Short: "Manage Kubernetes resources.",
		Long:  `This command provides access to Kubernetes features within Atlas.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			log.SetWriter(cmd.ErrOrStderr())
			if debugLevel {
				log.SetLevel(log.DebugLevel)
			}

			err := coreConfig.LoadAtlasCLIConfig()
			if err != nil {
				return fmt.Errorf("failed to load Atlas CLI configuration: %v", err)
			}
			if err := coreConfig.InitProfile(profile); err != nil {
				return fmt.Errorf("Failed to initialise Atlas CLI profile: %v", err)
			}

			telemetry.TrackPluginCommand(cmd, args)
			return nil
		},
	}

	cmd.AddCommand(config.Builder(), operator.Builder())

	cmd.PersistentFlags().StringVarP(&profile, flag.Profile, flag.ProfileShort, "", usage.ProfileAtlasCLI)
	cmd.PersistentFlags().BoolVarP(&debugLevel, flag.Debug, flag.DebugShort, false, usage.Debug)
	_ = cmd.PersistentFlags().MarkHidden(flag.Debug)

	return cmd
}
