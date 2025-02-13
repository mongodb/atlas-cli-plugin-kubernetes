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

package telemetry

import (
	"strings"
	"time"

	"github.com/mongodb/atlas-cli-core/config"
	"github.com/spf13/pflag"
)

type Event struct {
	Timestamp  time.Time      `json:"timestamp"`
	Source     string         `json:"source"`
	Properties map[string]any `json:"properties"`
}

type EventOpt func(Event)

type ConfigNameGetter interface {
	Name() string
}

type CmdName interface {
	CommandPath() string
	CalledAs() string
}

func withCommandPath(cmd CmdName) EventOpt {
	return func(event Event) {
		cmdPath := cmd.CommandPath()
		// remove the first character if it is " "
		if cmdPath != "" && cmdPath[0] == ' ' {
			cmdPath = cmdPath[1:]
		}

		event.Properties["command"] = strings.ReplaceAll(cmdPath, " ", "-")
		if cmd.CalledAs() != "" {
			event.Properties["alias"] = cmd.CalledAs()
		}
	}
}

func withEventType() EventOpt {
	return func(event Event) {
		event.Properties["eventType"] = "plugin"
	}
}

type CmdFlags interface {
	Flags() *pflag.FlagSet
}

func withFlags(cmd CmdFlags) EventOpt {
	return func(event Event) {
		setFlags := make([]string, 0, cmd.Flags().NFlag())
		cmd.Flags().Visit(func(f *pflag.Flag) {
			setFlags = append(setFlags, f.Name)
		})

		if len(setFlags) > 0 {
			event.Properties["flags"] = setFlags
		}
	}
}

func newEvent(opts ...EventOpt) Event {
	var event = Event{
		Timestamp: time.Now(),
		Source:    config.AtlasCLI,
		Properties: map[string]any{
			"result": "SUCCESS",
		},
	}

	for _, fn := range opts {
		fn(event)
	}

	return event
}
