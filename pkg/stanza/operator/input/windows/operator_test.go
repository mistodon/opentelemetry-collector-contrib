// Copyright  The OpenTelemetry Authors
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

//go:build windows
// +build windows

package windows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

func TestConfig(t *testing.T) {
	expect := NewConfig()

	input := map[string]interface{}{
		"id":            "windows_eventlog_input",
		"type":          "windows_eventlog_input",
		"max_reads":     100,
		"start_at":      "end",
		"poll_interval": time.Second,
		"attributes":    map[string]interface{}{},
		"resource":      map[string]interface{}{},
	}

	var actual Config
	err := helper.UnmarshalMapstructure(input, &actual)
	require.NoError(t, err)
	require.Equal(t, expect, &actual)
}
