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

//go:build e2e || install || generate || apply

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

const (
	maxRetryAttempts = 5
)

// atlasE2ETestGenerator is about providing capabilities to provide projects and clusters for our e2e tests.
type atlasE2ETestGenerator struct {
	projectID            string
	projectName          string
	clusterName          string
	clusterRegion        string
	flexName             string
	teamName             string
	teamID               string
	teamUser             string
	dbUser               string
	tier                 string
	mDBVer               string
	dataFedName          string
	streamInstanceName   string
	streamConnectionName string
	enableBackup         bool
	t                    *testing.T
}

// Logf formats its arguments according to the format, analogous to Printf, and
// records the text in the error log. A final newline is added if not provided. For
// tests, the text will be printed only if the test fails or the -test.v flag is
// set. For benchmarks, the text is always printed to avoid having performance
// depend on the value of the -test.v flag.
func (g *atlasE2ETestGenerator) Logf(format string, args ...any) {
	g.t.Logf(format, args...)
}

// newAtlasE2ETestGenerator creates a new instance of atlasE2ETestGenerator struct.
func newAtlasE2ETestGenerator(t *testing.T) *atlasE2ETestGenerator {
	t.Helper()
	return &atlasE2ETestGenerator{t: t}
}

func newAtlasE2ETestGeneratorWithBackup(t *testing.T) *atlasE2ETestGenerator {
	t.Helper()
	return &atlasE2ETestGenerator{t: t, enableBackup: true}
}

func (g *atlasE2ETestGenerator) generateTeam(prefix string) {
	g.t.Helper()

	if g.teamID != "" {
		g.t.Fatal("unexpected error: team was already generated")
	}

	var err error
	if prefix == "" {
		g.teamName, err = RandTeamName()
	} else {
		g.teamName, err = RandTeamNameWithPrefix(prefix)
	}
	if err != nil {
		g.t.Fatalf("unexpected error: %v", err)
	}

	g.teamUser, err = getFirstOrgUser()
	if err != nil {
		g.t.Fatalf("unexpected error retrieving org user: %v", err)
	}
	g.teamID, err = createTeam(g.teamName, g.teamUser)
	if err != nil {
		g.t.Fatalf("unexpected error creating team: %v", err)
	}
	g.Logf("teamID=%s", g.teamID)
	g.Logf("teamName=%s", g.teamName)
	if g.teamID == "" {
		g.t.Fatal("teamID not created")
	}
	g.t.Cleanup(func() {
		deleteTeamWithRetry(g.t, g.teamID)
	})
}

// generateProject generates a new project and also registers its deletion on test cleanup.
func (g *atlasE2ETestGenerator) generateProject(prefix string) {
	g.t.Helper()

	if g.projectID != "" {
		g.t.Fatal("unexpected error: project was already generated")
	}

	var err error
	if prefix == "" {
		g.projectName, err = RandProjectName()
	} else {
		g.projectName, err = RandProjectNameWithPrefix(prefix)
	}
	if err != nil {
		g.t.Fatalf("unexpected error: %v", err)
	}

	g.projectID, err = createProject(g.projectName)
	if err != nil {
		g.t.Fatalf("unexpected error creating project: %v", err)
	}
	g.Logf("projectID=%s", g.projectID)
	g.Logf("projectName=%s", g.projectName)
	if g.projectID == "" {
		g.t.Fatal("projectID not created")
	}

	g.t.Cleanup(func() {
		deleteProjectWithRetry(g.t, g.projectID)
	})
}

func (g *atlasE2ETestGenerator) generateEmptyProject(prefix string) {
	g.t.Helper()

	if g.projectID != "" {
		g.t.Fatal("unexpected error: project was already generated")
	}

	var err error
	if prefix == "" {
		g.projectName, err = RandProjectName()
	} else {
		g.projectName, err = RandProjectNameWithPrefix(prefix)
	}
	if err != nil {
		g.t.Fatalf("unexpected error: %v", err)
	}

	g.projectID, err = createProjectWithoutAlertSettings(g.projectName)
	if err != nil {
		g.t.Fatalf("unexpected error: %v", err)
	}
	g.t.Logf("projectID=%s", g.projectID)
	g.t.Logf("projectName=%s", g.projectName)
	if g.projectID == "" {
		g.t.Fatal("projectID not created")
	}

	g.t.Cleanup(func() {
		deleteProjectWithRetry(g.t, g.projectID)
	})
}

func (g *atlasE2ETestGenerator) generatePrivateEndpoint(provider, region string) {
	g.t.Helper()

	cliPath, err := AtlasCLIBin()
	if err != nil {
		g.t.Fatalf("%v: invalid bin", err)
	}

	cmd := exec.Command(cliPath,
		privateEndpointsEntity,
		provider,
		"create",
		"--region",
		region,
		"--projectId",
		g.projectID,
		"-o=json")
	cmd.Env = os.Environ()
	resp, err := test.RunAndGetStdOut(cmd)
	require.NoError(g.t, err, string(resp))
	var r atlasv2.EndpointService
	require.NoError(g.t, json.Unmarshal(resp, &r))

	g.t.Logf("endpointServiceID=%s", r.GetId())

	g.t.Cleanup(func() {
		g.t.Logf("deleting private endpoint service - ID=%s", r.GetId())
		cmd := exec.Command(cliPath,
			privateEndpointsEntity,
			provider,
			"delete",
			r.GetId(),
			"--projectId",
			g.projectID,
			"--force")
		cmd.Env = os.Environ()
		resp, err := test.RunAndGetStdOut(cmd)
		require.NoError(g.t, err, string(resp))

		cmd = exec.Command(cliPath,
			privateEndpointsEntity,
			provider,
			"watch",
			r.GetId(),
			"--projectId",
			g.projectID)
		cmd.Env = os.Environ()

		resp, err = cmd.CombinedOutput()
		// We expect a 404 error once the private endpoint has been completely deleted
		require.Error(g.t, err)
		assert.Contains(g.t, string(resp), "404")
	})
}

func (g *atlasE2ETestGenerator) generateDBUser(prefix string) {
	g.t.Helper()

	if g.projectID == "" {
		g.t.Fatal("unexpected error: project must be generated")
	}

	if g.dbUser != "" {
		g.t.Fatal("unexpected error: DBUser was already generated")
	}

	var err error
	if prefix == "" {
		g.dbUser, err = RandTeamName()
	} else {
		g.dbUser, err = RandTeamNameWithPrefix(prefix)
	}
	if err != nil {
		g.t.Fatalf("unexpected error: %v", err)
	}

	err = createDBUserWithCert(g.projectID, g.dbUser)
	if err != nil {
		g.dbUser = ""
		g.t.Fatalf("unexpected error: %v", err)
	}
	g.t.Logf("dbUser=%s", g.dbUser)
}

func deleteTeamWithRetry(t *testing.T, teamID string) {
	t.Helper()
	deleted := false
	backoff := 1
	for attempts := 1; attempts <= maxRetryAttempts; attempts++ {
		e := deleteTeam(teamID)
		if e == nil || strings.Contains(e.Error(), "GROUP_NOT_FOUND") {
			t.Logf("team %q successfully deleted", teamID)
			deleted = true
			break
		}
		t.Logf("%d/%d attempts - trying again in %d seconds: unexpected error while deleting the team %q: %v", attempts, maxRetryAttempts, backoff, teamID, e)
		time.Sleep(time.Duration(backoff) * time.Second)
		backoff *= 2
	}

	if !deleted {
		t.Errorf("we could not delete the team %q", teamID)
	}
}

func deleteProjectWithRetry(t *testing.T, projectID string) {
	t.Helper()
	deleted := false
	backoff := 1
	for attempts := 1; attempts <= maxRetryAttempts; attempts++ {
		e := deleteProject(projectID)
		if e == nil || strings.Contains(e.Error(), "GROUP_NOT_FOUND") {
			t.Logf("project %q successfully deleted", projectID)
			deleted = true
			break
		}
		t.Logf("%d/%d attempts - trying again in %d seconds: unexpected error while deleting the project %q: %v", attempts, maxRetryAttempts, backoff, projectID, e)
		time.Sleep(time.Duration(backoff) * time.Second)
		backoff *= 2
	}
	if !deleted {
		t.Errorf("we could not delete the project %q", projectID)
	}
}

func deleteKeys(t *testing.T, toDelete map[string]struct{}) {
	t.Helper()
	cliPath, err := AtlasCLIBin()

	cmd := exec.Command(cliPath,
		orgEntity,
		"apiKeys",
		"ls",
		"-o=json")

	cmd.Env = os.Environ()
	resp, err := test.RunAndGetStdOut(cmd)
	require.NoError(t, err, string(resp))

	var keys atlasv2.PaginatedApiApiUser
	err = json.Unmarshal(resp, &keys)
	require.NoError(t, err)

	uniqueKeysToDelete := make(map[string]struct{})
	for _, key := range keys.GetResults() {
		keyID := key.GetId()
		desc := key.GetDesc()

		if _, ok := toDelete[desc]; ok {
			uniqueKeysToDelete[keyID] = struct{}{}
		}
	}

	for keyID := range uniqueKeysToDelete {
		errors := []error{}
		t.Logf("Deleting key with ID: %s", keyID)
		cmd = exec.Command(cliPath,
			orgEntity,
			"apiKeys",
			"rm",
			keyID,
			"--force")
		cmd.Env = os.Environ()
		_, err = test.RunAndGetStdOutAndErr(cmd)
		if err != nil && !strings.Contains(err.Error(), "API_KEY_NOT_FOUND") {
			errors = append(errors, err)
		}
		if len(errors) > 0 {
			t.Errorf("unexpected errors while deleting keys: %v", errors)
		}
	}
}

func (g *atlasE2ETestGenerator) generateFlexCluster() {
	g.t.Helper()

	if g.projectID == "" {
		g.t.Fatal("unexpected error: project must be generated")
	}

	var err error
	g.flexName, err = deployFlexClusterForProject(g.projectID)
	if err != nil {
		g.t.Fatalf("unexpected error deploying flex cluster: %v", err)
	}
	g.t.Logf("flexClusterName=%s", g.flexName)

	g.t.Cleanup(func() {
		_ = deleteClusterForProject(g.projectID, g.flexName)
	})
}

// generateCluster generates a new cluster and also registers its deletion on test cleanup.
func (g *atlasE2ETestGenerator) generateCluster() {
	g.t.Helper()

	if g.projectID == "" {
		g.t.Fatal("unexpected error: project must be generated")
	}

	var err error
	if g.tier == "" {
		g.tier = e2eTier()
	}

	if g.mDBVer == "" {
		mdbVersion, e := MongoDBMajorVersion()
		require.NoError(g.t, e)

		g.mDBVer = mdbVersion
	}

	g.clusterName, g.clusterRegion, err = deployClusterForProject(g.projectID, g.tier, g.mDBVer, g.enableBackup)
	if err != nil {
		g.Logf("projectID=%q, clusterName=%q", g.projectID, g.clusterName)
		g.t.Errorf("unexpected error deploying cluster: %v", err)
	}
	g.t.Logf("clusterName=%s", g.clusterName)

	g.t.Cleanup(func() {
		g.Logf("Cluster cleanup %q\n", g.projectID)
		if e := deleteClusterForProject(g.projectID, g.clusterName); e != nil {
			g.t.Errorf("unexpected error deleting cluster: %v", e)
		}
	})
}

func (g *atlasE2ETestGenerator) generateDataFederation() {
	var err error
	g.t.Helper()

	if g.projectID == "" {
		g.t.Fatal("unexpected error: project must be generated")
	}

	g.dataFedName, err = createDataFederationForProject(g.projectID)
	storeName := g.dataFedName
	if err != nil {
		g.Logf("projectID=%q, dataFedName=%q", g.projectID, g.dataFedName)
		g.t.Errorf("unexpected error deploying data federation: %v", err)
	} else {
		g.Logf("dataFedName=%q", g.dataFedName)
	}

	g.t.Cleanup(func() {
		g.Logf("Data Federation cleanup %q\n", storeName)

		cliPath, err := AtlasCLIBin()
		require.NoError(g.t, err)

		deleteDataFederationForProject(g.t, cliPath, g.projectID, storeName)
		g.Logf("data federation %q successfully deleted", storeName)
	})
}

func (g *atlasE2ETestGenerator) generateStreamsInstance(name string) {
	g.t.Helper()

	if g.projectID == "" {
		g.t.Fatal("unexpected error: project must be generated")
	}

	var err error
	g.streamInstanceName, err = createStreamsInstance(g.t, g.projectID, name)
	instanceName := g.streamInstanceName
	if err != nil {
		g.Logf("projectID=%q, streamsInstanceName=%q", g.projectID, g.streamInstanceName)
		g.t.Errorf("unexpected error deploying streams instance: %v", err)
	} else {
		g.Logf("streamsInstanceName=%q", g.streamInstanceName)
	}

	g.t.Cleanup(func() {
		g.Logf("Streams instance cleanup %q\n", instanceName)

		require.NoError(g.t, deleteStreamsInstance(g.t, g.projectID, instanceName))
		g.Logf("streams instance %q successfully deleted", instanceName)
	})
}

func (g *atlasE2ETestGenerator) generateStreamsConnection(name string) {
	g.t.Helper()

	if g.projectID == "" {
		g.t.Fatal("unexpected error: project must be generated")
	}

	if g.streamInstanceName == "" {
		g.t.Fatal("unexpected error: streams instance must be generated")
	}

	var err error
	g.streamConnectionName, err = createStreamsConnection(g.t, g.projectID, g.streamInstanceName, name)
	connectionName := g.streamConnectionName
	if err != nil {
		g.Logf("projectID=%q, streamsConnectionName=%q", g.projectID, g.streamConnectionName)
		g.t.Errorf("unexpected error deploying streams instance: %v", err)
	} else {
		g.Logf("streamsConnectionName=%q", g.streamConnectionName)
	}

	g.t.Cleanup(func() {
		g.Logf("Streams connection cleanup %q\n", connectionName)

		require.NoError(g.t, deleteStreamsConnection(g.t, g.projectID, g.streamInstanceName, connectionName))
		g.Logf("streams connection %q successfully deleted", connectionName)
	})
}
