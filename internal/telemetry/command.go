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

package telemetry

import (
	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/log"
	"github.com/spf13/cobra"
)

type TrackOptions struct {
	Err    error
	Signal string
}

var currentTracker *pluginTracker
var options []EventOpt

func AppendOption(opt EventOpt) {
	options = append(options, opt)
}

func TrackPluginCommand(cmd *cobra.Command, args []string) {
	if !config.TelemetryEnabled() {
		return
	}
	var err error
	currentTracker, err = newTracker(cmd.Context(), cmd, args)
	if err != nil {
		_, _ = log.Debugf("telemetry: failed to create tracker: %v\n", err)
		return
	}

	if currentTracker == nil {
		return
	}

	if err := currentTracker.trackCommand(options...); err != nil {
		_, _ = log.Debugf("telemetry: failed to track command: %v\n", err)
	}
}
