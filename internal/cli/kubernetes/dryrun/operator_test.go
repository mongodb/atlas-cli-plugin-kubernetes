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
	"context"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"

	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWorkerRunSuccess(t *testing.T) {
	schm := scheme.Scheme
	assert.NoError(t, akov2.AddToScheme(schm))
	k8sClient := fake.NewClientBuilder().WithScheme(schm).WithRuntimeObjects().Build()
	worker := NewWorker().WithK8SClient(k8sClient).
		WithTargetNamespace("test").
		WithWatchNamespaces("test").
		WithOperatorVersion("v2.7.1").
		WithWaitTimeoutSec(10).
		WithWaitForCompletion(true)
	go func() {
		attempts := 3
		delay := 500 * time.Millisecond
		for {
			if attempts == 0 {
				assert.Failf(t, "failed to patch the job after 3 attempts", "")
			}
			var jobs batchv1.JobList
			assert.NoError(t, k8sClient.List(context.Background(), &jobs, client.MatchingLabels{"app": "ako-dry-run"}))
			if len(jobs.Items) != 1 {
				attempts -= 1
				time.Sleep(delay)
				continue
			}
			job := jobs.Items[0]
			job.Status.Succeeded = 1
			assert.NoError(t, k8sClient.Status().Update(context.Background(), &job))
			return
		}
	}()
	assert.NoError(t, worker.Run())
}

func TestWorkerFailure(t *testing.T) {
	schm := scheme.Scheme
	assert.NoError(t, akov2.AddToScheme(schm))
	k8sClient := fake.NewClientBuilder().WithScheme(schm).WithRuntimeObjects().Build()
	worker := NewWorker().WithK8SClient(k8sClient).
		WithTargetNamespace("test").
		WithWatchNamespaces("test").
		WithOperatorVersion("v2.7.1").
		WithWaitTimeoutSec(10).
		WithWaitForCompletion(true)
	assert.Error(t, worker.Run())
}
