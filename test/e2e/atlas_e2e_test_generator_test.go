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

//  //go:build e2e || install || generate || apply

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/test"
	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/atlas-sdk/v20241113004/admin"
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

const (
	maxRetryAttempts = 5
)

const (
	googleSAFilename = ".googleServiceAccount.json"
)

type awsVPC struct {
	id     string
	region string
	cidr   string
}

type azureClient struct {
	resourceGroupName      string
	credentials            *azidentity.DefaultAzureCredential
	networkResourceFactory *armnetwork.ClientFactory
}

type azureVNet struct {
	id     string
	name   string
	region string
	cidr   string
}

type gcpConnection struct {
	projectID string

	networkClient *compute.NetworksClient
}

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
	containerID          string
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

	cmd = exec.Command(cliPath,
		networkingEntity,
		networkContainerEntity,
		"list",
		"--projectId",
		g.projectID,
		"-o=json")
	cmd.Env = os.Environ()
	resp, err = test.RunAndGetStdOut(cmd)
	require.NoError(g.t, err, string(resp))
	var nc []atlasv2.CloudProviderContainer
	require.NoError(g.t, json.Unmarshal(resp, &nc))
	g.containerID = nc[0].GetId()

	g.t.Logf("containerID=%s", nc[0].GetId())

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

func (g *atlasE2ETestGenerator) generateAWSContainer(cidr, region string) string {
	g.t.Helper()

	client := MustGetNewTestClientFromEnv(g.t)
	ctx := context.Background()
	containerRequest := &atlasv2.CloudProviderContainer{
		ProviderName:   pointer.Get(string(akov2provider.ProviderAWS)),
		AtlasCidrBlock: pointer.Get(cidr),
		RegionName:     pointer.Get(region),
	}
	createdContainer, _, err := client.NetworkPeeringApi.CreatePeeringContainer(ctx, g.projectID, containerRequest).Execute()
	if err != nil {
		g.t.Fatalf("failed to create test container: %v", err)
	}
	return createdContainer.GetId()
}

func (g *atlasE2ETestGenerator) generateAzureContainer(cidr, region string) string {
	g.t.Helper()

	client := MustGetNewTestClientFromEnv(g.t)
	ctx := context.Background()
	containerRequest := &atlasv2.CloudProviderContainer{
		ProviderName:   pointer.Get(string(akov2provider.ProviderAzure)),
		AtlasCidrBlock: pointer.Get(cidr),
		Region:         pointer.Get(region),
	}
	createdContainer, _, err := client.NetworkPeeringApi.CreatePeeringContainer(ctx, g.projectID, containerRequest).Execute()
	if err != nil {
		g.t.Fatalf("failed to create test container: %v", err)
	}
	return createdContainer.GetId()
}

func (g *atlasE2ETestGenerator) generateGCPContainer(cidr string) string {
	g.t.Helper()

	client := MustGetNewTestClientFromEnv(g.t)
	ctx := context.Background()
	containerRequest := &atlasv2.CloudProviderContainer{
		ProviderName:   pointer.Get(string(akov2provider.ProviderGCP)),
		AtlasCidrBlock: pointer.Get(cidr),
	}
	createdContainer, _, err := client.NetworkPeeringApi.CreatePeeringContainer(ctx, g.projectID, containerRequest).Execute()
	if err != nil {
		g.t.Fatalf("failed to create test container: %v", err)
	}
	return createdContainer.GetId()
}

func (g *atlasE2ETestGenerator) generateAWSNetworkVPC(cidr, region string) *awsVPC {
	g.t.Helper()

	name := vpcName(g.t, region)

	awsSession, err := newAWSSession()
	if err != nil {
		g.t.Fatalf("failed to create an AWS session: %v", err)
	}
	ec2Client := ec2.New(awsSession, aws.NewConfig().WithRegion(region))

	result, err := ec2Client.CreateVpc(&ec2.CreateVpcInput{
		AmazonProvidedIpv6CidrBlock: aws.Bool(false),
		CidrBlock:                   aws.String(cidr),
		TagSpecifications: []*ec2.TagSpecification{{
			ResourceType: aws.String(ec2.ResourceTypeVpc),
			Tags: []*ec2.Tag{
				{Key: aws.String("Name"), Value: aws.String(name)},
			},
		}},
	})
	if err != nil {
		g.t.Fatalf("failed to create AWS test VPC in %s: %v", region, err)
	}
	if result == nil {
		g.t.Fatalf("failed to create AWS test VPC in %s: no result", region)
	}
	if result.Vpc == nil {
		g.t.Fatalf("failed to create AWS test VPC in %s: no VPC populated", region)
	}
	if result.Vpc.VpcId == nil {
		g.t.Fatalf("failed to create AWS test VPC in %s: no VPC ID set", region)
	}
	vpcId := *result.Vpc.VpcId

	_, err = ec2Client.ModifyVpcAttribute(&ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		VpcId: &vpcId,
	})
	if err != nil {
		g.t.Fatalf("failed to modify test VPC %q in %s: %v", vpcId, region, err)
	}

	return &awsVPC{id: vpcId, cidr: cidr, region: region}
}

func (g *atlasE2ETestGenerator) deleteAWSNetworkVPC(vpc *awsVPC) {
	g.t.Helper()

	awsSession, err := newAWSSession()
	if err != nil {
		g.t.Fatalf("failed to create an AWS session: %v", err)
	}
	ec2Client := ec2.New(awsSession, aws.NewConfig().WithRegion(vpc.region))

	input := &ec2.DeleteVpcInput{
		DryRun: aws.Bool(false),
		VpcId:  aws.String(vpc.id),
	}

	if _, err := ec2Client.DeleteVpc(input); err != nil {
		g.t.Fatalf("failed to delete test VPC %q in %s: %v", vpc.id, vpc.region, err)
	}
}

func newAWSSession() (*session.Session, error) {
	if _, ok := os.LookupEnv("AWS_ACCESS_KEY_ID"); !ok {
		return nil, errors.New("missing env var AWS_ACCESS_KEY_ID")
	}
	if _, ok := os.LookupEnv("AWS_SECRET_ACCESS_KEY"); !ok {
		return nil, errors.New("missing env var AWS_SECRET_ACCESS_KEY")
	}
	return session.NewSession(aws.NewConfig())
}

func (g *atlasE2ETestGenerator) generateAWSPeering(containerID string, vpc *awsVPC) string {
	g.t.Helper()
	return g.generatePeering(&atlasv2.BaseNetworkPeeringConnectionSettings{
		ContainerId:         containerID,
		ProviderName:        pointer.Get(string(akov2provider.ProviderAWS)),
		AccepterRegionName:  pointer.Get(vpc.region),
		AwsAccountId:        pointer.Get(os.Getenv("AWS_ACCOUNT_ID")),
		RouteTableCidrBlock: pointer.Get(vpc.cidr),
		VpcId:               pointer.Get(vpc.id),
	})
}

func (g *atlasE2ETestGenerator) generateAzurePeering(containerID string, vnet *azureVNet) string {
	g.t.Helper()
	return g.generatePeering(&atlasv2.BaseNetworkPeeringConnectionSettings{
		ContainerId:         containerID,
		ProviderName:        pointer.Get(string(akov2provider.ProviderAzure)),
		AzureSubscriptionId: pointer.Get(os.Getenv("AZURE_SUBSCRIPTION_ID")),
		AzureDirectoryId:    pointer.Get(os.Getenv("AZURE_TENANT_ID")),
		ResourceGroupName:   pointer.Get(os.Getenv("AZURE_RESOURCE_GROUP_NAME")),
		VnetName:            pointer.Get(vnet.name),
	})
}

func (g *atlasE2ETestGenerator) generateGCPPeering(containerID string, networkName string) string {
	g.t.Helper()
	return g.generatePeering(&atlasv2.BaseNetworkPeeringConnectionSettings{
		ContainerId:  containerID,
		ProviderName: pointer.Get(string(akov2provider.ProviderGCP)),
		GcpProjectId: pointer.Get(os.Getenv("GOOGLE_PROJECT_ID")),
		NetworkName:  pointer.Get(networkName),
	})
}

func (g *atlasE2ETestGenerator) generateAzureVPC(cidr, region string) *azureVNet {
	g.t.Helper()
	azr, err := newAzureClient()
	if err != nil {
		g.t.Fatalf("failed to create azure client: %v", err)
	}

	subnetsSpec := []*armnetwork.Subnet{
		{
			Name: pointer.Get("default-subnet"),
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: pointer.Get(cidr),
			},
		},
	}

	vpcClient := azr.networkResourceFactory.NewVirtualNetworksClient()
	ctx := context.Background()
	vpcName := vpcName(g.t, region)
	op, err := vpcClient.BeginCreateOrUpdate(
		ctx,
		azr.resourceGroupName,
		vpcName,
		armnetwork.VirtualNetwork{
			Location: pointer.Get(region),
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{
						pointer.Get(cidr),
					},
				},
				Subnets: subnetsSpec,
			},
			Tags: map[string]*string{
				"Name": pointer.Get(vpcName),
			},
		},
		nil,
	)
	if err != nil {
		g.t.Fatalf("failed to begin create azure VPC: %v", err)
	}

	vpc, err := op.PollUntilDone(ctx, nil)
	if err != nil {
		g.t.Fatalf("creation process of Azure VPC failed: %v", err)
	}
	if vpc.Name == nil {
		g.t.Fatal("VPC created without a name")
	}
	if vpc.ID == nil {
		g.t.Fatal("VPC created without ID")
	}
	return &azureVNet{id: *vpc.ID, name: *vpc.Name, cidr: cidr, region: region}
}

func (g *atlasE2ETestGenerator) deleteAzureVPC(vpc *azureVNet) {
	g.t.Helper()
	azr, err := newAzureClient()
	if err != nil {
		g.t.Fatalf("failed to create azure client: %v", err)
	}
	vpcClient := azr.networkResourceFactory.NewVirtualNetworksClient()
	ctx := context.Background()
	op, err := vpcClient.BeginDelete(
		ctx,
		azr.resourceGroupName,
		vpc.name,
		nil,
	)
	if err != nil {
		g.t.Fatalf("Failed to delete Azure VPC: %v", err)
	}

	if _, err = op.PollUntilDone(ctx, nil); err != nil {
		g.t.Fatalf("Failed to check Azure VPC was deleted: %v", err)
	}
}

func newAzureClient() (*azureClient, error) {
	if _, ok := os.LookupEnv("AZURE_CLIENT_ID"); !ok {
		return nil, errors.New("missing env var AZURE_CLIENT_ID")
	}
	if _, ok := os.LookupEnv("AZURE_TENANT_ID"); !ok {
		return nil, errors.New("missing env var AZURE_TENANT_ID")
	}
	if _, ok := os.LookupEnv("AZURE_CLIENT_SECRET"); !ok {
		return nil, errors.New("missing env var AZURE_CLIENT_SECRET")
	}
	rg, ok := os.LookupEnv("AZURE_RESOURCE_GROUP_NAME")
	if !ok {
		return nil, errors.New("missing env var AZURE_RESOURCE_GROUP_NAME")
	}
	subscriptionID, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		return nil, errors.New("missing env var AZURE_SUBSCRIPTION_ID")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	networkFactory, err := armnetwork.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return &azureClient{
		resourceGroupName:      rg,
		networkResourceFactory: networkFactory,
		credentials:            cred,
	}, err
}

func vpcName(t *testing.T, region string) string {
	return strings.ToLower(fmt.Sprintf("cli-plugin-test-vpc-at-%s-%s", region, randomString(t)))
}

func (g *atlasE2ETestGenerator) generateGCPNetworkVPC() string {
	ctx := context.Background()
	gcp, err := newGCPConnection(ctx, os.Getenv("GOOGLE_PROJECT_ID"))
	if err != nil {
		g.t.Fatalf("failed to get Google Cloud connection: %v", err)
	}
	vpcName := vpcName(g.t, "google")

	op, err := gcp.networkClient.Insert(ctx, &computepb.InsertNetworkRequest{
		Project: gcp.projectID,
		NetworkResource: &computepb.Network{
			Name:                  pointer.Get(vpcName),
			Description:           pointer.Get("Atlas Kubernetes CLI plugin E2E Tests VPC"),
			AutoCreateSubnetworks: pointer.Get(false),
		},
	})
	if err != nil {
		g.t.Fatalf("failed to request creation of Google VPC: %v", err)
	}

	err = op.Wait(ctx)
	if err != nil {
		g.t.Fatalf("failed to create Google VPC: %v", err)
	}

	return vpcName
}

func (g *atlasE2ETestGenerator) deleteGCPNetworkVPC(vpcName string) {
	ctx := context.Background()
	gcp, err := newGCPConnection(ctx, os.Getenv("GOOGLE_PROJECT_ID"))
	if err != nil {
		g.t.Fatalf("failed to get Google Cloud connection: %v", err)
	}
	op, err := gcp.networkClient.Delete(ctx, &computepb.DeleteNetworkRequest{
		Project: gcp.projectID,
		Network: vpcName,
	})
	if err != nil {
		g.t.Fatalf("failed to request deletion of Google VPC: %v", err)
	}
	err = op.Wait(ctx)
	if err != nil {
		g.t.Fatalf("failed to delete Google VPC: %v", err)
	}
}

func newGCPConnection(ctx context.Context, projectID string) (*gcpConnection, error) {
	if err := ensureGCPCredentials(); err != nil {
		return nil, fmt.Errorf("failed to prepare credentials")
	}

	networkClient, err := compute.NewNetworksRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup network rest client")
	}

	return &gcpConnection{
		projectID:     projectID,
		networkClient: networkClient,
	}, nil
}

func ensureGCPCredentials() error {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		return nil
	}
	credentials := os.Getenv("GCP_SA_CRED")
	if credentials == "" {
		return errors.New("GOOGLE_APPLICATION_CREDENTIALS and GCP_SA_CRED are unset, cant setup Google credentials")
	}
	if err := os.WriteFile(googleSAFilename, ([]byte)(credentials), 0600); err != nil {
		return fmt.Errorf("failed to save credentials contents GCP_SA_CRED to %s: %w",
			googleSAFilename, err)
	}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", googleSAFilename)
	return nil
}

func (g *atlasE2ETestGenerator) generatePeering(request *atlasv2.BaseNetworkPeeringConnectionSettings) string {
	client := MustGetNewTestClientFromEnv(g.t)
	ctx := context.Background()
	createdPeering, _, err := client.NetworkPeeringApi.CreatePeeringConnection(ctx, g.projectID, request).Execute()
	if err != nil {
		g.t.Fatalf("failed to create test peering: %v", err)
	}
	return createdPeering.GetId()
}

func (g *atlasE2ETestGenerator) deletePeering(id string) {
	g.t.Helper()

	client := MustGetNewTestClientFromEnv(g.t)
	ctx := context.Background()
	_, _, err := client.NetworkPeeringApi.DeletePeeringConnection(ctx, g.projectID, id).Execute()
	if err != nil {
		g.t.Fatalf("failed to delete test peering %s: %v", id, err)
	}
	start := time.Now()
	pause := time.Second
	timeout := 5 * time.Minute
	for {
		_, _, err := client.NetworkPeeringApi.GetPeeringConnection(ctx, g.projectID, id).Execute()
		if admin.IsErrorCode(err, "PEER_NOT_FOUND") {
			return
		}
		if err != nil {
			g.t.Fatalf("failed to check deletion of test peering %s: %v", id, err)
		}
		if time.Since(start) > timeout {
			g.t.Fatalf("timed out checking for deletion of test peering %s: %v", id, err)
		}
		time.Sleep(pause)
		pause = pause * 2
	}
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
