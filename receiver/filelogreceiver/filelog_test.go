// Copyright The OpenTelemetry Authors
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

package filelogreceiver

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/observiq/nanojack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
)

func TestDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, configtest.CheckConfigStruct(cfg))
}

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(config.NewComponentID("filelog").String())
	require.NoError(t, err)
	require.NoError(t, config.UnmarshalReceiver(sub, cfg))

	assert.NoError(t, cfg.Validate())
	assert.Equal(t, testdataConfigYaml(), cfg)
}

func TestCreateWithInvalidInputConfig(t *testing.T) {
	t.Parallel()

	cfg := testdataConfigYaml()
	cfg.InputConfig.StartAt = "middle"

	_, err := NewFactory().CreateLogsReceiver(
		context.Background(),
		componenttest.NewNopReceiverCreateSettings(),
		cfg,
		new(consumertest.LogsSink),
	)
	require.Error(t, err, "receiver creation should fail if given invalid input config")
}

func TestReadStaticFile(t *testing.T) {
	t.Parallel()

	expectedTimestamp, _ := time.ParseInLocation("2006-01-02", "2020-08-25", time.Local)

	f := NewFactory()
	sink := new(consumertest.LogsSink)

	cfg := testdataConfigYaml()
	cfg.Converter.MaxFlushCount = 10
	cfg.Converter.FlushInterval = time.Millisecond

	converter := adapter.NewConverter()
	converter.Start()
	defer converter.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go consumeNLogsFromConverter(converter.OutChannel(), 3, &wg)

	rcvr, err := f.CreateLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, sink)
	require.NoError(t, err, "failed to create receiver")
	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()))

	// Build the expected set by using adapter.Converter to translate entries
	// to pdata Logs.
	queueEntry := func(t *testing.T, c *adapter.Converter, msg string, severity entry.Severity) {
		e := entry.New()
		e.Timestamp = expectedTimestamp
		require.NoError(t, e.Set(entry.NewBodyField("msg"), msg))
		e.Severity = severity
		e.AddAttribute("file_name", "simple.log")
		require.NoError(t, c.Batch([]*entry.Entry{e}))
	}
	queueEntry(t, converter, "Something routine", entry.Info)
	queueEntry(t, converter, "Something bad happened!", entry.Error)
	queueEntry(t, converter, "Some details...", entry.Debug)

	dir, err := os.Getwd()
	require.NoError(t, err)
	t.Logf("Working Directory: %s", dir)

	wg.Wait()

	require.Eventually(t, expectNLogs(sink, 3), 2*time.Second, 5*time.Millisecond,
		"expected %d but got %d logs",
		3, sink.LogRecordCount(),
	)
	// TODO: Figure out a nice way to assert each logs entry content.
	// require.Equal(t, expectedLogs, sink.AllLogs())
	require.NoError(t, rcvr.Shutdown(context.Background()))
}

func TestReadRotatingFiles(t *testing.T) {

	tests := []rotationTest{
		{
			name:         "CopyTruncateTimestamped",
			copyTruncate: true,
			sequential:   false,
		},
		{
			name:         "MoveCreateTimestamped",
			copyTruncate: false,
			sequential:   false,
		},
		{
			name:         "CopyTruncateSequential",
			copyTruncate: true,
			sequential:   true,
		},
		{
			name:         "MoveCreateSequential",
			copyTruncate: false,
			sequential:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.Run)
	}
}

type rotationTest struct {
	name         string
	copyTruncate bool
	sequential   bool
}

func (rt *rotationTest) Run(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	f := NewFactory()
	sink := new(consumertest.LogsSink)

	cfg := rotationTestConfig(tempDir)
	cfg.Converter.MaxFlushCount = 1
	cfg.Converter.FlushInterval = time.Millisecond

	// With a max of 100 logs per file and 1 backup file, rotation will occur
	// when more than 100 logs are written, and deletion when more than 200 are written.
	// Write 300 and validate that we got the all despite rotation and deletion.
	logger := newRotatingLogger(t, tempDir, 100, 1, rt.copyTruncate, rt.sequential)
	numLogs := 300

	// Build expected outputs
	expectedTimestamp, _ := time.ParseInLocation("2006-01-02", "2020-08-25", time.Local)
	converter := adapter.NewConverter()
	converter.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	go consumeNLogsFromConverter(converter.OutChannel(), numLogs, &wg)

	rcvr, err := f.CreateLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), cfg, sink)
	require.NoError(t, err, "failed to create receiver")
	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()))

	for i := 0; i < numLogs; i++ {
		msg := fmt.Sprintf("This is a simple log line with the number %3d", i)

		// Build the expected set by converting entries to pdata Logs...
		e := entry.New()
		e.Timestamp = expectedTimestamp
		require.NoError(t, e.Set(entry.NewBodyField("msg"), msg))
		require.NoError(t, converter.Batch([]*entry.Entry{e}))

		// ... and write the logs lines to the actual file consumed by receiver.
		logger.Printf("2020-08-25 %s", msg)
		time.Sleep(time.Millisecond)
	}

	wg.Wait()
	require.Eventually(t, expectNLogs(sink, numLogs), 2*time.Second, 10*time.Millisecond,
		"expected %d but got %d logs",
		numLogs, sink.LogRecordCount(),
	)
	// TODO: Figure out a nice way to assert each logs entry content.
	// require.Equal(t, expectedLogs, sink.AllLogs())
	require.NoError(t, rcvr.Shutdown(context.Background()))
	converter.Stop()
}

func consumeNLogsFromConverter(ch <-chan plog.Logs, count int, wg *sync.WaitGroup) {
	defer wg.Done()

	n := 0
	for pLog := range ch {
		n += pLog.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().Len()

		if n == count {
			return
		}
	}
}

func newRotatingLogger(t *testing.T, tempDir string, maxLines, maxBackups int, copyTruncate, sequential bool) *log.Logger {
	path := filepath.Join(tempDir, "test.log")
	rotator := &nanojack.Logger{
		Filename:     path,
		MaxLines:     maxLines,
		MaxBackups:   maxBackups,
		CopyTruncate: copyTruncate,
		Sequential:   sequential,
	}

	t.Cleanup(func() { _ = rotator.Close() })

	return log.New(rotator, "", 0)
}

func expectNLogs(sink *consumertest.LogsSink, expected int) func() bool {
	return func() bool { return sink.LogRecordCount() == expected }
}

func testdataConfigYaml() *FileLogConfig {
	return &FileLogConfig{
		BaseConfig: adapter.BaseConfig{
			ReceiverSettings: config.NewReceiverSettings(config.NewComponentID(typeStr)),
			Operators: adapter.OperatorConfigs{
				map[string]interface{}{
					"type":  "regex_parser",
					"regex": "^(?P<time>\\d{4}-\\d{2}-\\d{2}) (?P<sev>[A-Z]*) (?P<msg>.*)$",
					"severity": map[string]interface{}{
						"parse_from": "attributes.sev",
					},
					"timestamp": map[string]interface{}{
						"layout":     "%Y-%m-%d",
						"parse_from": "attributes.time",
					},
				},
			},
			Converter: adapter.ConverterConfig{
				MaxFlushCount: 100,
				FlushInterval: 100 * time.Millisecond,
			},
		},
		InputConfig: func() file.Config {
			c := file.NewConfig()
			c.Include = []string{"testdata/simple.log"}
			c.StartAt = "beginning"
			return *c
		}(),
	}
}

func rotationTestConfig(tempDir string) *FileLogConfig {
	return &FileLogConfig{
		BaseConfig: adapter.BaseConfig{
			ReceiverSettings: config.NewReceiverSettings(config.NewComponentID(typeStr)),
			Operators: adapter.OperatorConfigs{
				map[string]interface{}{
					"type":  "regex_parser",
					"regex": "^(?P<ts>\\d{4}-\\d{2}-\\d{2}) (?P<msg>[^\n]+)",
					"timestamp": map[interface{}]interface{}{
						"layout":     "%Y-%m-%d",
						"parse_from": "body.ts",
					},
				},
			},
			Converter: adapter.ConverterConfig{},
		},
		InputConfig: func() file.Config {
			c := file.NewConfig()
			c.Include = []string{fmt.Sprintf("%s/*", tempDir)}
			c.StartAt = "beginning"
			c.PollInterval = 10 * time.Millisecond
			c.IncludeFileName = false
			return *c
		}(),
	}
}
