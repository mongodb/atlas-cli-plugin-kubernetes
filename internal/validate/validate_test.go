// Copyright 2020 MongoDB Inc
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


package validate

import (
	"testing"
)

func TestObjectID(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{
			name:    "Empty value",
			val:     "",
			wantErr: true,
		},
		{
			name:    "Valid ObjectID",
			val:     "5e9f088b4797476aa0a5d56a",
			wantErr: false,
		},
		{
			name:    "Short ObjectID",
			val:     "5e9f088b4797476aa0a5d56",
			wantErr: true,
		},
		{
			name:    "Invalid ObjectID",
			val:     "5e9f088b4797476aa0a5d56z",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		val := tt.val
		wantErr := tt.wantErr
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ObjectID(val); (err != nil) != wantErr {
				t.Errorf("OptionalObjectID() error = %v, wantErr %v", err, wantErr)
			}
		})
	}
}
