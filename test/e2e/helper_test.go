// Copyright 2025 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//go:build e2e || kubernetes

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/test"
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
	"go.mongodb.org/atlas/mongodbatlas"
)

const (
	clustersEntity = "clusters"
	projectEntity  = "project"
	orgEntity      = "org"
)

const (
	e2eClusterTier            = "M10"
	e2eGovClusterTier         = "M20"
	e2eDefaultClusterProvider = "AWS"
)

func deployClusterForProject(projectID, tier, mDBVersion string, enableBackup bool) (string, string, error) {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return "", "", err
	}
	clusterName, err := RandClusterName()
	if err != nil {
		return "", "", err
	}
	region, err := newAvailableRegion(projectID, tier, e2eDefaultClusterProvider)
	if err != nil {
		return "", "", err
	}
	args := []string{
		clustersEntity,
		"create",
		clusterName,
		"--mdbVersion", mDBVersion,
		"--region", region,
		"--tier", tier,
		"--provider", e2eDefaultClusterProvider,
		"--diskSizeGB=30",
	}
	if enableBackup {
		args = append(args, "--backup")
	}
	if projectID != "" {
		args = append(args, "--projectId", projectID)
	}
	create := exec.Command(cliPath, args...)
	create.Env = os.Environ()
	if resp, err := test.RunAndGetStdOut(create); err != nil {
		return "", "", fmt.Errorf("error creating cluster %w: %s", err, string(resp))
	}

	watchArgs := []string{
		clustersEntity,
		"watch",
		clusterName,
	}
	if projectID != "" {
		watchArgs = append(watchArgs, "--projectId", projectID)
	}
	watch := exec.Command(cliPath, watchArgs...)
	watch.Env = os.Environ()
	if resp, err := test.RunAndGetStdOut(watch); err != nil {
		return "", "", fmt.Errorf("error watching cluster %w: %s", err, string(resp))
	}
	return clusterName, region, nil
}

func e2eTier() string {
	tier := e2eClusterTier
	if IsGov() {
		tier = e2eGovClusterTier
	}
	return tier
}

func internalDeleteClusterForProject(projectID, clusterName string) error {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return err
	}
	args := []string{
		clustersEntity,
		"delete",
		clusterName,
		"--force",
	}
	if projectID != "" {
		args = append(args, "--projectId", projectID)
	}
	deleteCmd := exec.Command(cliPath, args...)
	deleteCmd.Env = os.Environ()
	if resp, err := test.RunAndGetStdOut(deleteCmd); err != nil {
		return fmt.Errorf("error deleting cluster %w: %s", err, string(resp))
	}

	// this command will fail with 404 once the cluster is deleted
	// we just need to wait for this to close the project
	_ = watchCluster(projectID, clusterName)
	return nil
}

func watchCluster(projectID, clusterName string) error {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return err
	}
	watchArgs := []string{
		clustersEntity,
		"watch",
		clusterName,
	}
	if projectID != "" {
		watchArgs = append(watchArgs, "--projectId", projectID)
	}
	watchCmd := exec.Command(cliPath, watchArgs...)
	watchCmd.Env = os.Environ()
	if resp, err := test.RunAndGetStdOut(watchCmd); err != nil {
		return fmt.Errorf("error waiting for cluster %w: %s", err, string(resp))
	}
	return nil
}

func removeTerminationProtectionFromCluster(projectID, clusterName string) error {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return err
	}
	args := []string{
		clustersEntity,
		"update",
		clusterName,
		"--disableTerminationProtection",
	}
	if projectID != "" {
		args = append(args, "--projectId", projectID)
	}
	updateCmd := exec.Command(cliPath, args...)
	updateCmd.Env = os.Environ()
	if resp, err := test.RunAndGetStdOut(updateCmd); err != nil {
		return fmt.Errorf("error updating cluster %w: %s", err, string(resp))
	}

	return watchCluster(projectID, clusterName)
}

func deleteClusterForProject(projectID, clusterName string) error {
	if err := internalDeleteClusterForProject(projectID, clusterName); err != nil {
		if !strings.Contains(err.Error(), "CANNOT_TERMINATE_CLUSTER_WHEN_TERMINATION_PROTECTION_ENABLED") {
			return err
		}
		if err := removeTerminationProtectionFromCluster(projectID, clusterName); err != nil {
			return err
		}
		return internalDeleteClusterForProject(projectID, clusterName)
	}

	return nil
}

var errNoRegions = errors.New("no regions available")

func newAvailableRegion(projectID, tier, provider string) (string, error) {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return "", err
	}
	args := []string{
		clustersEntity,
		"availableRegions",
		"ls",
		"--provider", provider,
		"--tier", tier,
		"-o=json",
	}
	if projectID != "" {
		args = append(args, "--projectId", projectID)
	}
	cmd := exec.Command(cliPath, args...)
	cmd.Env = os.Environ()
	resp, err := test.RunAndGetStdOut(cmd)

	if err != nil {
		return "", fmt.Errorf("error getting regions %w: %s", err, string(resp))
	}

	var cloudProviders atlasv2.PaginatedApiAtlasProviderRegions
	err = json.Unmarshal(resp, &cloudProviders)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response %w: %s", err, string(resp))
	}

	if cloudProviders.GetTotalCount() == 0 || len(cloudProviders.GetResults()[0].GetInstanceSizes()) == 0 {
		return "", errNoRegions
	}

	return cloudProviders.GetResults()[0].GetInstanceSizes()[0].GetAvailableRegions()[0].GetName(), nil
}

func RandClusterName() (string, error) {
	n, err := RandInt(1000)
	if err != nil {
		return "", err
	}
	if revision, ok := os.LookupEnv("revision"); ok {
		return fmt.Sprintf("cluster-%v-%s", n, revision), nil
	}
	return fmt.Sprintf("cluster-%v", n), nil
}

func RandProjectName() (string, error) {
	n, err := RandInt(1000)
	if err != nil {
		return "", err
	}
	if revision, ok := os.LookupEnv("revision"); ok {
		return fmt.Sprintf("%v-%s", n, revision), nil
	}
	return fmt.Sprintf("e2e-%v", n), nil
}

func RandProjectNameWithPrefix(prefix string) (string, error) {
	name, err := RandProjectName()
	if err != nil {
		return "", err
	}
	prefixedName := fmt.Sprintf("%s-%s", prefix, name)
	if len(prefixedName) > 64 {
		return prefixedName[:64], nil
	}
	return prefixedName, nil
}

func MongoDBMajorVersion() (string, error) {
	atlasClient := mongodbatlas.NewClient(nil)
	atlasURL := os.Getenv("MCLI_OPS_MANAGER_URL")
	baseURL, err := url.Parse(atlasURL)
	if err != nil {
		return "", err
	}
	atlasClient.BaseURL = baseURL
	version, _, err := atlasClient.DefaultMongoDBMajorVersion.Get(context.Background())
	if err != nil {
		return "", err
	}

	return version, nil
}

func IsGov() bool {
	return os.Getenv("MCLI_SERVICE") == "cloudgov"
}

func createProject(projectName string) (string, error) {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return "", fmt.Errorf("%w: invalid bin", err)
	}
	args := []string{
		projectEntity,
		"create",
		projectName,
		"-o=json",
	}
	if IsGov() {
		args = append(args, "--govCloudRegionsOnly")
	}
	cmd := exec.Command(cliPath, args...)
	cmd.Env = os.Environ()
	resp, err := test.RunAndGetStdOut(cmd)
	if err != nil {
		return "", fmt.Errorf("%s (%w)", string(resp), err)
	}

	var project atlasv2.Group
	if err := json.Unmarshal(resp, &project); err != nil {
		return "", fmt.Errorf("invalid response: %s (%w)", string(resp), err)
	}

	return project.GetId(), nil
}

func deleteProject(projectID string) error {
	cliPath, err := AtlasCLIBin()
	if err != nil {
		return err
	}
	cmd := exec.Command(cliPath,
		projectEntity,
		"delete",
		projectID,
		"--force")
	cmd.Env = os.Environ()
	resp, err := test.RunAndGetStdOut(cmd)
	if err != nil {
		return fmt.Errorf("%s (%w)", string(resp), err)
	}
	return nil
}

func prepareK8sName(pattern string) string {
	return resources.NormalizeAtlasName(pattern, resources.AtlasNameToKubernetesName())
}
