// Copyright 2023 MongoDB Inc
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

//go:build unit

package operator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallOptsValidateIpAccessList(t *testing.T) {
	tests := map[string]struct {
		ipAccessList string
		err          error
	}{
		"valid single IP": {
			ipAccessList: "104.30.164.5",
		},
		"valid CIDR block": {
			ipAccessList: "192.168.100.177/24",
		},
		"valid list of entries": {
			ipAccessList: "104.30.164.5,192.168.100.177/24",
		},
		"empty string": {
			ipAccessList: "",
			err:          errors.New("IP access list cannot be empty"),
		},
		"invalid IP": {
			ipAccessList: "256.256.256.256",
			err:          errors.New("IP access list \"256.256.256.256\" must be a valid IP address or CIDR"),
		},
		"invalid CIDR block": {
			ipAccessList: "192.168.100.177/33",
			err:          errors.New("IP access list \"192.168.100.177/33\" must be a valid IP address or CIDR"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			opts := &InstallOpts{
				ipAccessList: tt.ipAccessList,
			}
			err := opts.ValidateIpAccessList()
			assert.Equal(t, tt.err, err)
		})
	}
}
