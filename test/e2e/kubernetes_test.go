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

//go:build e2e || apply || generate || install

package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	runner := m.Run
	if isQASelected() {
		runner = func() int {
			setQACredentialsEnvVars()
			defer restoreEnvVars()
			return m.Run()
		}
	}
	os.Exit(runner())
}

func TestEnv(t *testing.T) {
	if isQASelected() {
		t.Run("Switched to run on cloud-qa", func(_ *testing.T) {})
	} else {
		t.Run("Running on cloud-dev (default)", func(_ *testing.T) {})
	}
}

func isQASelected() bool {
	return os.Getenv("ATLAS_TEST_ENV") == "QA" && areQASettingsPresent()
}

func isQAEnv(managerURL string) bool {
	return strings.Contains(managerURL, "cloud-qa.mongodb.com")
}

func areQASettingsPresent() bool {
	return hasEnv("QA_MCLI_ORG_ID") &&
		hasEnv("QA_MCLI_OPS_MANAGER_URL") &&
		hasEnv("QA_MCLI_PUBLIC_API_KEY") &&
		hasEnv("QA_MCLI_PRIVATE_API_KEY")
}

func hasEnv(envvar string) bool {
	_, ok := os.LookupEnv(envvar)
	return ok
}

func setQACredentialsEnvVars() {
	saveEnv("MCLI_ORG_ID")
	saveEnv("MCLI_OPS_MANAGER_URL")
	saveEnv("MCLI_PUBLIC_API_KEY")
	saveEnv("MCLI_PRIVATE_API_KEY")
	os.Setenv("MCLI_ORG_ID", os.Getenv("QA_MCLI_ORG_ID"))
	os.Setenv("MCLI_OPS_MANAGER_URL", os.Getenv("QA_MCLI_OPS_MANAGER_URL"))
	os.Setenv("MCLI_PUBLIC_API_KEY", os.Getenv("QA_MCLI_PUBLIC_API_KEY"))
	os.Setenv("MCLI_PRIVATE_API_KEY", os.Getenv("QA_MCLI_PRIVATE_API_KEY"))
}

func restoreEnvVars() {
	restoreEnv("MCLI_ORG_ID")
	restoreEnv("MCLI_OPS_MANAGER_URL")
	restoreEnv("MCLI_PUBLIC_API_KEY")
	restoreEnv("MCLI_PRIVATE_API_KEY")
}

func saveEnv(envvar string) {
	os.Setenv(savedEnvVar(envvar), os.Getenv(envvar))
}

func restoreEnv(envvar string) {
	os.Setenv(envvar, os.Getenv(savedEnvVar(envvar)))
}

func savedEnvVar(envvar string) string {
	return fmt.Sprintf("%s_SAVED", envvar)
}
