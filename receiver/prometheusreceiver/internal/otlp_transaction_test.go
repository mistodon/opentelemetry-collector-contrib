// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/obsreport"
)

func Test_transaction_pdata(t *testing.T) {
	// discoveredLabels contain labels prior to any processing
	discoveredLabels := labels.New(
		labels.Label{
			Name:  model.AddressLabel,
			Value: "address:8080",
		},
		labels.Label{
			Name:  model.MetricNameLabel,
			Value: "foo",
		},
		labels.Label{
			Name:  model.SchemeLabel,
			Value: "http",
		},
	)
	// processedLabels contain label values after processing (e.g. relabeling)
	processedLabels := labels.New(
		labels.Label{
			Name:  model.InstanceLabel,
			Value: "localhost:8080",
		},
	)

	target := scrape.NewTarget(processedLabels, discoveredLabels, nil)
	scrapeCtx := scrape.ContextWithTarget(context.Background(), target)
	scrapeCtx = scrape.ContextWithMetricMetadataStore(scrapeCtx, noopMetricMetadataStore{})

	t.Run("Commit Without Adding", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(scrapeCtx, nil, true, "", nomc, nil, componenttest.NewNopReceiverCreateSettings(), nopObsRecv())
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
	})

	t.Run("Rollback does nothing", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(scrapeCtx, nil, true, "", nomc, nil, componenttest.NewNopReceiverCreateSettings(), nopObsRecv())
		if got := tr.Rollback(); got != nil {
			t.Errorf("expecting nil from Rollback() but got err %v", got)
		}
	})

	badLabels := labels.Labels([]labels.Label{{Name: "foo", Value: "bar"}})
	t.Run("Add One No Target", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(scrapeCtx, nil, true, "", nomc, nil, componenttest.NewNopReceiverCreateSettings(), nopObsRecv())
		if _, got := tr.Append(0, badLabels, time.Now().Unix()*1000, 1.0); got == nil {
			t.Errorf("expecting error from Add() but got nil")
		}
	})

	jobNotFoundLb := labels.Labels([]labels.Label{
		{Name: "instance", Value: "localhost:8080"},
		{Name: "job", Value: "test2"},
		{Name: "foo", Value: "bar"}})
	t.Run("Add One Job not found", func(t *testing.T) {
		nomc := consumertest.NewNop()
		tr := newTransaction(scrapeCtx, nil, true, "", nomc, nil, componenttest.NewNopReceiverCreateSettings(), nopObsRecv())
		if _, got := tr.Append(0, jobNotFoundLb, time.Now().Unix()*1000, 1.0); got == nil {
			t.Errorf("expecting error from Add() but got nil")
		}
	})

	goodLabels := labels.Labels([]labels.Label{{Name: "instance", Value: "localhost:8080"},
		{Name: "job", Value: "test"},
		{Name: "__name__", Value: "foo"}})
	t.Run("Add One Good", func(t *testing.T) {
		sink := new(consumertest.MetricsSink)
		tr := newTransaction(scrapeCtx, nil, true, "", sink, nil, componenttest.NewNopReceiverCreateSettings(), nopObsRecv())
		if _, got := tr.Append(0, goodLabels, time.Now().Unix()*1000, 1.0); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		tr.metricBuilder.startTime = 1.0 // set to a non-zero value
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
		l := []labels.Label{{Name: "__scheme__", Value: "http"}}
		expectedNodeResource := CreateResource("test", "localhost:8080", l)
		mds := sink.AllMetrics()
		if len(mds) != 1 {
			t.Fatalf("wanted one batch, got %v\n", sink.AllMetrics())
		}
		gotNodeResource := mds[0].ResourceMetrics().At(0).Resource()
		require.Equal(t, expectedNodeResource, gotNodeResource, "Resources do not match")
		// TODO: re-enable this when handle unspecified OC type
		// assert.Len(t, ocmds[0].Metrics, 1)
	})

	t.Run("Error when start time is zero", func(t *testing.T) {
		sink := new(consumertest.MetricsSink)
		tr := newTransaction(scrapeCtx, nil, true, "", sink, nil, componenttest.NewNopReceiverCreateSettings(), nopObsRecv())
		if _, got := tr.Append(0, goodLabels, time.Now().Unix()*1000, 1.0); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		tr.metricBuilder.startTime = 0 // zero value means the start time metric is missing
		got := tr.Commit()
		if got == nil {
			t.Error("expecting error from Commit() but got nil")
		} else if got.Error() != errNoStartTimeMetrics.Error() {
			t.Errorf("expected error %q but got %q", errNoStartTimeMetrics, got)
		}
	})
}

func nopObsRecv() *obsreport.Receiver {
	return obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             config.NewComponentID("prometheus"),
		Transport:              transport,
		ReceiverCreateSettings: componenttest.NewNopReceiverCreateSettings(),
	})
}

type noopMetricMetadataStore struct{}

func (noopMetricMetadataStore) ListMetadata() []scrape.MetricMetadata { return nil }
func (noopMetricMetadataStore) GetMetadata(metric string) (scrape.MetricMetadata, bool) {
	return scrape.MetricMetadata{}, false
}
func (noopMetricMetadataStore) SizeMetadata() int   { return 0 }
func (noopMetricMetadataStore) LengthMetadata() int { return 0 }
