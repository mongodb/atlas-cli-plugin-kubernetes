// Copyright 2025 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

package dryrun

import (
	"fmt"
	"strings"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/cli/require"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/flag"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/usage"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var ErrUnsupportedOperatorVersionFmt = "version %q is not supported. Supported versions: %v"

const defaultTimeoutSec = 120

type Opts struct {
	cli.OrgOpts
	cli.OutputOpts

	operatorVersion string
	targetNamespace string
	watchNamespaces []string
	waitForJob      bool
	waitTimeout     int64
}

func (opts *Opts) ValidateTargetNamespace() error {
	if errs := validation.IsDNS1123Label(opts.targetNamespace); len(errs) != 0 {
		return fmt.Errorf("%s parameter is invalid: %v", flag.OperatorTargetNamespace, errs)
	}
	return nil
}

func (opts *Opts) ValidateOperatorVersion() error {
	if _, versionFound := features.GetResourcesForVersion(opts.operatorVersion); versionFound {
		return nil
	}
	return fmt.Errorf(ErrUnsupportedOperatorVersionFmt, opts.operatorVersion, features.SupportedVersions())
}

func (opts *Opts) MakeK8SClient() (client.Client, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s config: %w", err)
	}

	c, err := client.New(conf, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return c, nil
}

func (opts *Opts) Run() error {
	k8sClient, err := opts.MakeK8SClient()
	if err != nil {
		return err
	}

	worker := NewWorker().
		WithTargetNamespace(opts.targetNamespace).
		WithWatchNamespaces(strings.Join(opts.watchNamespaces, ",")).
		WithOperatorVersion(opts.operatorVersion).
		WithWaitForCompletion(opts.waitForJob).
		WithWaitTimeoutSec(opts.waitTimeout).
		WithK8SClient(k8sClient)
	return worker.Run()
}

// Builder builds a cobra.Command for the Kubernetes dryrun installation.
func Builder() *cobra.Command {
	const use = "dry-run"

	opts := &Opts{}

	cmd := &cobra.Command{
		Use:     use,
		Args:    require.NoArgs,
		Aliases: cli.GenerateAliases(use),
		Short:   "Deploy and run Atlas Kubernetes Operator in dry-run mode",
		Long:    `This command deploys the Atlas Kubernetes operator with the DryRun mode.`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return opts.PreRunE(
				opts.ValidateTargetNamespace,
				opts.ValidateOperatorVersion,
			)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.Run()
		},
	}

	opts.AddOrgOptFlags(cmd)
	cmd.Flags().StringVar(&opts.targetNamespace, flag.OperatorTargetNamespace, "", usage.OperatorTargetNamespace)
	cmd.Flags().StringSliceVar(&opts.watchNamespaces, flag.OperatorWatchNamespace, []string{}, usage.OperatorWatchNamespace)
	cmd.Flags().StringVar(&opts.operatorVersion, flag.OperatorVersion, features.LatestOperatorMajorVersion, usage.OperatorVersion)
	cmd.Flags().BoolVar(&opts.waitForJob, flag.EnableWatch, false, usage.EnableWatch)
	cmd.Flags().Int64Var(&opts.waitTimeout, flag.WatchTimeout, defaultTimeoutSec, usage.WatchTimeout)
	return cmd
}
