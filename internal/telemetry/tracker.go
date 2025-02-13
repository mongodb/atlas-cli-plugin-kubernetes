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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/log"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	cacheFilename           = "telemetry"
	dirPermissions          = 0700
	filePermissions         = 0600
	defaultMaxCacheFileSize = 500_000 // 500KB
)

type pluginTracker struct {
	fs               afero.Fs
	maxCacheFileSize int64
	cacheDir         string
	cmd              *cobra.Command
	args             []string
}

func newTracker(ctx context.Context, cmd *cobra.Command, args []string) (*pluginTracker, error) {
	var err error

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	cacheDir = filepath.Join(cacheDir, config.AtlasCLI)

	t := &pluginTracker{
		fs:               afero.NewOsFs(),
		maxCacheFileSize: defaultMaxCacheFileSize,
		cacheDir:         cacheDir,
		cmd:              cmd,
		args:             args,
	}

	return t, nil
}

func (t *pluginTracker) trackCommand(opt ...EventOpt) error {
	o := append(
		[]EventOpt{},
		withCommandPath(t.cmd),
		withFlags(t.cmd),
		withEventType())

	o = append(o, opt...)
	event := newEvent(o...)
	events, err := t.read()
	if err != nil {
		_, _ = log.Debugf("telemetry: failed to read cache: %v\n", err)
	}
	events = append(events, event)
	_, _ = log.Debugf("telemetry: events: %v\n", events)

	return t.save(event)
}

// Read all events in the cache file.
func (t *pluginTracker) read() ([]Event, error) {
	initialSize := 100
	events := make([]Event, 0, initialSize)
	filename := filepath.Join(t.cacheDir, cacheFilename)
	exists, err := afero.Exists(t.fs, filename)
	if err != nil {
		return events, err
	}
	if exists {
		file, err := t.fs.Open(filename)
		if err != nil {
			return events, err
		}
		defer file.Close()
		decoder := json.NewDecoder(file)
		for decoder.More() {
			var event Event
			if err := decoder.Decode(&event); err != nil {
				return events, err
			}
			events = append(events, event)
		}
	}
	return events, nil
}

// Append a single event to the cache file.
func (t *pluginTracker) save(event Event) error {
	file, err := t.openCacheFile()
	if err != nil {
		return err
	}
	fmt.Println(file)

	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = file.Write(data)
	return err
}

func (t *pluginTracker) openCacheFile() (afero.File, error) {
	exists, err := afero.DirExists(t.fs, t.cacheDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		if mkdirError := t.fs.MkdirAll(t.cacheDir, dirPermissions); mkdirError != nil {
			return nil, mkdirError
		}
	}
	filename := filepath.Join(t.cacheDir, cacheFilename)
	exists, err = afero.Exists(t.fs, filename)
	if err != nil {
		return nil, err
	}
	if exists {
		info, statError := t.fs.Stat(filename)
		if statError != nil {
			return nil, statError
		}
		size := info.Size()
		if size > t.maxCacheFileSize {
			return nil, errors.New("telemetry cache file too large")
		}
	}
	file, err := t.fs.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, filePermissions)
	return file, err
}
