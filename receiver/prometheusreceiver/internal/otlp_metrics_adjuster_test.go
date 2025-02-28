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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

var (
	tUnknown = pdataTimestampFromMs(0)
	t1       = pdataTimestampFromMs(1)
	t2       = pdataTimestampFromMs(2)
	t3       = pdataTimestampFromMs(3)
	t4       = pdataTimestampFromMs(4)
	t5       = pdataTimestampFromMs(5)

	bounds0  = []float64{1, 2, 4}
	percent0 = []float64{10, 50, 90}

	sum1       = "sum1"
	gauge1     = "gauge1"
	histogram1 = "histogram1"
	summary1   = "summary1"

	k1v1k2v2 = []*kv{
		{"k1", "v1"},
		{"k2", "v2"},
	}

	k1v10k2v20 = []*kv{
		{"k1", "v10"},
		{"k2", "v20"},
	}

	k1v100k2v200 = []*kv{
		{"k1", "v100"},
		{"k2", "v200"},
	}

	emptyLabels              = []*kv{}
	k1vEmpty                 = []*kv{{"k1", ""}}
	k1vEmptyk2vEmptyk3vEmpty = []*kv{{"k1", ""}, {"k2", ""}, {"k3", ""}}
)

func Test_gauge_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Gauge: round 1 - gauge not adjusted",
			metrics:     metricSlice(gaugeMetric(gauge1, doublePoint(k1v1k2v2, t1, t1, 44))),
			adjusted:    metricSlice(gaugeMetric(gauge1, doublePoint(k1v1k2v2, t1, t1, 44))),
		},
		{
			description: "Gauge: round 2 - gauge not adjusted",
			metrics:     metricSlice(gaugeMetric(gauge1, doublePoint(k1v1k2v2, t2, t2, 66))),
			adjusted:    metricSlice(gaugeMetric(gauge1, doublePoint(k1v1k2v2, t2, t2, 66))),
		},
		{
			description: "Gauge: round 3 - value less than previous value - gauge is not adjusted",
			metrics:     metricSlice(gaugeMetric(gauge1, doublePoint(k1v1k2v2, t3, t3, 55))),
			adjusted:    metricSlice(gaugeMetric(gauge1, doublePoint(k1v1k2v2, t3, t3, 55))),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_sum_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Sum: round 1 - initial instance, start time is established",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44))),
		},
		{
			description: "Sum: round 2 - instance adjusted based on round 1",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t2, t2, 66))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t1, t2, 66))),
		},
		{
			description: "Sum: round 3 - instance reset (value less than previous value), start time is reset",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t3, t3, 55))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t3, t3, 55))),
		},
		{
			description: "Sum: round 4 - instance adjusted based on round 3",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t4, t4, 72))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t3, t4, 72))),
		},
		{
			description: "Sum: round 5 - instance adjusted based on round 4",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t5, t5, 72))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t3, t5, 72))),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_summary_no_count_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Summary No Count: round 1 - initial instance, start time is established",
			metrics:     metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 0, 40, percent0, []float64{1, 5, 8}))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 0, 40, percent0, []float64{1, 5, 8}))),
		},
		{
			description: "Summary No Count: round 2 - instance adjusted based on round 1",
			metrics:     metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t2, t2, 0, 70, percent0, []float64{7, 44, 9}))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t2, 0, 70, percent0, []float64{7, 44, 9}))),
		},
		{
			description: "Summary No Count: round 3 - instance reset (count less than previous), start time is reset",
			metrics:     metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t3, 0, 66, percent0, []float64{3, 22, 5}))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t3, 0, 66, percent0, []float64{3, 22, 5}))),
		},
		{
			description: "Summary No Count: round 4 - instance adjusted based on round 3",
			metrics:     metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t4, t4, 0, 96, percent0, []float64{9, 47, 8}))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t4, 0, 96, percent0, []float64{9, 47, 8}))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_summary_flag_norecordedvalue(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Summary No Count: round 1 - initial instance, start time is established",
			metrics:     metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 0, 40, percent0, []float64{1, 5, 8}))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 0, 40, percent0, []float64{1, 5, 8}))),
		},
		{
			description: "Summary Flag NoRecordedValue: round 2 - instance adjusted based on round 1",
			metrics:     metricSlice(summaryMetric(summary1, summaryPointNoValue(k1v1k2v2, t2, t2))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPointNoValue(k1v1k2v2, t1, t2))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_summary_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Summary: round 1 - initial instance, start time is established",
			metrics: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 10, 40, percent0, []float64{1, 5, 8})),
			),
			adjusted: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 10, 40, percent0, []float64{1, 5, 8})),
			),
		},
		{
			description: "Summary: round 2 - instance adjusted based on round 1",
			metrics: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t2, t2, 15, 70, percent0, []float64{7, 44, 9})),
			),
			adjusted: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t2, 15, 70, percent0, []float64{7, 44, 9})),
			),
		},
		{
			description: "Summary: round 3 - instance reset (count less than previous), start time is reset",
			metrics: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t3, 12, 66, percent0, []float64{3, 22, 5})),
			),
			adjusted: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t3, 12, 66, percent0, []float64{3, 22, 5})),
			),
		},
		{
			description: "Summary: round 4 - instance adjusted based on round 3",
			metrics: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t4, t4, 14, 96, percent0, []float64{9, 47, 8})),
			),
			adjusted: metricSlice(
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t4, 14, 96, percent0, []float64{9, 47, 8})),
			),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_histogram_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Histogram: round 1 - initial instance, start time is established",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7}))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7}))),
		}, {
			description: "Histogram: round 2 - instance adjusted based on round 1",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t2, t2, bounds0, []uint64{6, 3, 4, 8}))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t2, bounds0, []uint64{6, 3, 4, 8}))),
		}, {
			description: "Histogram: round 3 - instance reset (value less than previous value), start time is reset",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t3, bounds0, []uint64{5, 3, 2, 7}))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t3, bounds0, []uint64{5, 3, 2, 7}))),
		}, {
			description: "Histogram: round 4 - instance adjusted based on round 3",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t4, t4, bounds0, []uint64{7, 4, 2, 12}))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t4, bounds0, []uint64{7, 4, 2, 12}))),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_histogram_flag_norecordedvalue(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Histogram: round 1 - initial instance, start time is established",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{7, 4, 2, 12}))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{7, 4, 2, 12}))),
		},
		{
			description: "Histogram: round 2 - instance adjusted based on round 1",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPointNoValue(k1v1k2v2, tUnknown, t2))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPointNoValue(k1v1k2v2, t1, t2))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_histogram_flag_norecordedvalue_first_observation(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Histogram: round 1 - initial instance, start time is unknown",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPointNoValue(k1v1k2v2, tUnknown, t1))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPointNoValue(k1v1k2v2, tUnknown, t1))),
		},
		{
			description: "Histogram: round 2 - instance unchanged",
			metrics:     metricSlice(histogramMetric(histogram1, histogramPointNoValue(k1v1k2v2, tUnknown, t2))),
			adjusted:    metricSlice(histogramMetric(histogram1, histogramPointNoValue(k1v1k2v2, tUnknown, t2))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_summary_flag_norecordedvalue_first_observation(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Summary: round 1 - initial instance, start time is unknown",
			metrics:     metricSlice(summaryMetric(summary1, summaryPointNoValue(k1v1k2v2, tUnknown, t1))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPointNoValue(k1v1k2v2, tUnknown, t1))),
		},
		{
			description: "Summary: round 2 - instance unchanged",
			metrics:     metricSlice(summaryMetric(summary1, summaryPointNoValue(k1v1k2v2, tUnknown, t2))),
			adjusted:    metricSlice(summaryMetric(summary1, summaryPointNoValue(k1v1k2v2, tUnknown, t2))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_gauge_flag_norecordedvalue_first_observation(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Gauge: round 1 - initial instance, start time is unknown",
			metrics:     metricSlice(gaugeMetric(gauge1, doublePointNoValue(k1v1k2v2, tUnknown, t1))),
			adjusted:    metricSlice(gaugeMetric(gauge1, doublePointNoValue(k1v1k2v2, tUnknown, t1))),
		},
		{
			description: "Gauge: round 2 - instance unchanged",
			metrics:     metricSlice(gaugeMetric(gauge1, doublePointNoValue(k1v1k2v2, tUnknown, t2))),
			adjusted:    metricSlice(gaugeMetric(gauge1, doublePointNoValue(k1v1k2v2, tUnknown, t2))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_sum_flag_norecordedvalue_first_observation(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "Sum: round 1 - initial instance, start time is unknown",
			metrics:     metricSlice(sumMetric("sum1", doublePointNoValue(k1v1k2v2, tUnknown, t1))),
			adjusted:    metricSlice(sumMetric("sum1", doublePointNoValue(k1v1k2v2, tUnknown, t1))),
		},
		{
			description: "Sum: round 2 - instance unchanged",
			metrics:     metricSlice(sumMetric("sum1", doublePointNoValue(k1v1k2v2, tUnknown, t2))),
			adjusted:    metricSlice(sumMetric("sum1", doublePointNoValue(k1v1k2v2, tUnknown, t2))),
		},
	}

	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_multiMetrics_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "MultiMetrics: round 1 - combined round 1 of individual metrics",
			metrics: metricSlice(
				gaugeMetric(gauge1, doublePoint(k1v1k2v2, t1, t1, 44)),
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 10, 40, percent0, []float64{1, 5, 8})),
			),
			adjusted: metricSlice(
				gaugeMetric(gauge1, doublePoint(k1v1k2v2, t1, t1, 44)),
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t1, 10, 40, percent0, []float64{1, 5, 8})),
			),
		},
		{
			description: "MultiMetrics: round 2 - combined round 2 of individual metrics",
			metrics: metricSlice(
				gaugeMetric(gauge1, doublePoint(k1v1k2v2, t2, t2, 66)),
				sumMetric(sum1, doublePoint(k1v1k2v2, t2, t2, 66)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t2, t2, bounds0, []uint64{6, 3, 4, 8})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t2, t2, 15, 70, percent0, []float64{7, 44, 9})),
			),
			adjusted: metricSlice(
				gaugeMetric(gauge1, doublePoint(k1v1k2v2, t2, t2, 66)),
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t2, 66)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t2, bounds0, []uint64{6, 3, 4, 8})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t1, t2, 15, 70, percent0, []float64{7, 44, 9})),
			),
		},
		{
			description: "MultiMetrics: round 3 - combined round 3 of individual metrics",
			metrics: metricSlice(
				gaugeMetric(gauge1, doublePoint(k1v1k2v2, t3, t3, 55)),
				sumMetric(sum1, doublePoint(k1v1k2v2, t3, t3, 55)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t3, bounds0, []uint64{5, 3, 2, 7})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t3, 12, 66, percent0, []float64{3, 22, 5})),
			),
			adjusted: metricSlice(
				gaugeMetric(gauge1, doublePoint(k1v1k2v2, t3, t3, 55)),
				sumMetric(sum1, doublePoint(k1v1k2v2, t3, t3, 55)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t3, bounds0, []uint64{5, 3, 2, 7})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t3, 12, 66, percent0, []float64{3, 22, 5})),
			),
		},
		{
			description: "MultiMetrics: round 4 - combined round 4 of individual metrics",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t4, t4, 72)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t4, t4, bounds0, []uint64{7, 4, 2, 12})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t4, t4, 14, 96, percent0, []float64{9, 47, 8})),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t3, t4, 72)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t4, bounds0, []uint64{7, 4, 2, 12})),
				summaryMetric(summary1, summaryPoint(k1v1k2v2, t3, t4, 14, 96, percent0, []float64{9, 47, 8})),
			),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_new_datapoints_added(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "New Datapoints: round 1 - two datapoints each",
			metrics: metricSlice(
				sumMetric(sum1,
					doublePoint(k1v1k2v2, t1, t1, 44),
					doublePoint(k1v100k2v200, t1, t1, 44)),
				histogramMetric(histogram1,
					histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7}),
					histogramPoint(k1v100k2v200, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				summaryMetric(summary1,
					summaryPoint(k1v1k2v2, t1, t1, 10, 40, percent0, []float64{1, 5, 8}),
					summaryPoint(k1v100k2v200, t1, t1, 10, 40, percent0, []float64{1, 5, 8})),
			),
			adjusted: metricSlice(
				sumMetric(sum1,
					doublePoint(k1v1k2v2, t1, t1, 44),
					doublePoint(k1v100k2v200, t1, t1, 44)),
				histogramMetric(histogram1,
					histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7}),
					histogramPoint(k1v100k2v200, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				summaryMetric(summary1,
					summaryPoint(k1v1k2v2, t1, t1, 10, 40, percent0, []float64{1, 5, 8}),
					summaryPoint(k1v100k2v200, t1, t1, 10, 40, percent0, []float64{1, 5, 8})),
			),
		},
		{
			description: "New Datapoints: round 2 - new datapoints unchanged, old datapoints adjusted",
			metrics: metricSlice(
				sumMetric(sum1,
					doublePoint(k1v1k2v2, t2, t2, 44),
					doublePoint(k1v10k2v20, t2, t2, 44),
					doublePoint(k1v100k2v200, t2, t2, 44)),
				histogramMetric(histogram1,
					histogramPoint(k1v1k2v2, t2, t2, bounds0, []uint64{4, 2, 3, 7}),
					histogramPoint(k1v10k2v20, t2, t2, bounds0, []uint64{4, 2, 3, 7}),
					histogramPoint(k1v100k2v200, t2, t2, bounds0, []uint64{4, 2, 3, 7})),
				summaryMetric(summary1,
					summaryPoint(k1v1k2v2, t2, t2, 10, 40, percent0, []float64{1, 5, 8}),
					summaryPoint(k1v10k2v20, t2, t2, 10, 40, percent0, []float64{1, 5, 8}),
					summaryPoint(k1v100k2v200, t2, t2, 10, 40, percent0, []float64{1, 5, 8})),
			),
			adjusted: metricSlice(
				sumMetric(sum1,
					doublePoint(k1v1k2v2, t1, t2, 44),
					doublePoint(k1v10k2v20, t2, t2, 44),
					doublePoint(k1v100k2v200, t1, t2, 44)),
				histogramMetric(histogram1,
					histogramPoint(k1v1k2v2, t1, t2, bounds0, []uint64{4, 2, 3, 7}),
					histogramPoint(k1v10k2v20, t2, t2, bounds0, []uint64{4, 2, 3, 7}),
					histogramPoint(k1v100k2v200, t1, t2, bounds0, []uint64{4, 2, 3, 7})),
				summaryMetric(summary1,
					summaryPoint(k1v1k2v2, t1, t2, 10, 40, percent0, []float64{1, 5, 8}),
					summaryPoint(k1v10k2v20, t2, t2, 10, 40, percent0, []float64{1, 5, 8}),
					summaryPoint(k1v100k2v200, t1, t2, 10, 40, percent0, []float64{1, 5, 8})),
			),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_multiTimeseries_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "MultiTimeseries: round 1 - initial first instance, start time is established",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44))),
		},
		{
			description: "MultiTimeseries: round 2 - first instance adjusted based on round 1, initial second instance",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t2, t2, 66)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t2, t2, 20.0)),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t2, 66)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t2, t2, 20.0)),
			),
		},
		{
			description: "MultiTimeseries: round 3 - first instance adjusted based on round 1, second based on round 2",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t3, t3, 88.0)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t3, t3, 49.0)),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t3, 88.0)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t2, t3, 49.0)),
			),
		},
		{
			description: "MultiTimeseries: round 4 - first instance reset, second instance adjusted based on round 2, initial third instance",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t4, t4, 87.0)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t4, t4, 57.0)),
				sumMetric(sum1, doublePoint(k1v100k2v200, t4, t4, 10.0)),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t4, t4, 87.0)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t2, t4, 57.0)),
				sumMetric(sum1, doublePoint(k1v100k2v200, t4, t4, 10.0)),
			),
		},
		{
			description: "MultiTimeseries: round 5 - first instance adjusted based on round 4, second on round 2, third on round 4",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t5, t5, 90.0)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t5, t5, 65.0)),
				sumMetric(sum1, doublePoint(k1v100k2v200, t5, t5, 22.0)),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t4, t5, 90.0)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t2, t5, 65.0)),
				sumMetric(sum1, doublePoint(k1v100k2v200, t4, t5, 22.0)),
			),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_emptyLabels_pdata(t *testing.T) {
	script := []*metricsAdjusterTest{
		{
			description: "EmptyLabels: round 1 - initial instance, implicitly empty labels, start time is established",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(emptyLabels, t1, t1, 44))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(emptyLabels, t1, t1, 44))),
		},
		{
			description: "EmptyLabels: round 2 - instance adjusted based on round 1",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(emptyLabels, t2, t2, 66))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(emptyLabels, t1, t2, 66))),
		},
		{
			description: "EmptyLabels: round 3 - one explicitly empty label, instance adjusted based on round 1",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1vEmpty, t3, t3, 77))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1vEmpty, t1, t3, 77))),
		},
		{
			description: "EmptyLabels: round 4 - three explicitly empty labels, instance adjusted based on round 1",
			metrics:     metricSlice(sumMetric(sum1, doublePoint(k1vEmptyk2vEmptyk3vEmpty, t3, t3, 88))),
			adjusted:    metricSlice(sumMetric(sum1, doublePoint(k1vEmptyk2vEmptyk3vEmpty, t1, t3, 88))),
		},
	}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_tsGC_pdata(t *testing.T) {
	script1 := []*metricsAdjusterTest{
		{
			description: "TsGC: round 1 - initial instances, start time is established",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t1, t1, 20)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t1, t1, bounds0, []uint64{40, 20, 30, 70})),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t1, t1, 20)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t1, t1, bounds0, []uint64{40, 20, 30, 70})),
			),
		},
	}

	script2 := []*metricsAdjusterTest{
		{
			description: "TsGC: round 2 - metrics first timeseries adjusted based on round 2, second timeseries not updated",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t2, t2, 88)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t2, t2, bounds0, []uint64{8, 7, 9, 14})),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t2, 88)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t2, bounds0, []uint64{8, 7, 9, 14})),
			),
		},
	}

	script3 := []*metricsAdjusterTest{
		{
			description: "TsGC: round 3 - metrics first timeseries adjusted based on round 2, second timeseries empty due to timeseries gc()",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t3, t3, 99)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t3, t3, 80)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t3, t3, bounds0, []uint64{9, 8, 10, 15})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t3, t3, bounds0, []uint64{55, 66, 33, 77})),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t3, 99)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t3, t3, 80)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t3, bounds0, []uint64{9, 8, 10, 15})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t3, t3, bounds0, []uint64{55, 66, 33, 77})),
			),
		},
	}

	jobsMap := NewJobsMap(time.Minute)

	// run round 1
	runScript(t, jobsMap.get("job", "0"), script1)
	// gc the tsmap, unmarking all entries
	jobsMap.get("job", "0").gc()
	// run round 2 - update metrics first timeseries only
	runScript(t, jobsMap.get("job", "0"), script2)
	// gc the tsmap, collecting umarked entries
	jobsMap.get("job", "0").gc()
	// run round 3 - verify that metrics second timeseries have been gc'd
	runScript(t, jobsMap.get("job", "0"), script3)
}

func Test_jobGC_pdata(t *testing.T) {
	job1Script1 := []*metricsAdjusterTest{
		{
			description: "JobGC: job 1, round 1 - initial instances, adjusted should be empty",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t1, t1, 20)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t1, t1, bounds0, []uint64{40, 20, 30, 70})),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t1, t1, 44)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t1, t1, 20)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t1, t1, bounds0, []uint64{4, 2, 3, 7})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t1, t1, bounds0, []uint64{40, 20, 30, 70})),
			),
		},
	}

	job2Script1 := []*metricsAdjusterTest{
		{
			description: "JobGC: job2, round 1 - no metrics adjusted, just trigger gc",
			metrics:     pmetric.NewMetricSlice(),
			adjusted:    pmetric.NewMetricSlice(),
		},
	}

	job1Script2 := []*metricsAdjusterTest{
		{
			description: "JobGC: job 1, round 2 - metrics timeseries empty due to job-level gc",
			metrics: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t4, t4, 99)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t4, t4, 80)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t4, t4, bounds0, []uint64{9, 8, 10, 15})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t4, t4, bounds0, []uint64{55, 66, 33, 77})),
			),
			adjusted: metricSlice(
				sumMetric(sum1, doublePoint(k1v1k2v2, t4, t4, 99)),
				sumMetric(sum1, doublePoint(k1v10k2v20, t4, t4, 80)),
				histogramMetric(histogram1, histogramPoint(k1v1k2v2, t4, t4, bounds0, []uint64{9, 8, 10, 15})),
				histogramMetric(histogram1, histogramPoint(k1v10k2v20, t4, t4, bounds0, []uint64{55, 66, 33, 77})),
			),
		},
	}

	gcInterval := 10 * time.Millisecond
	jobsMap := NewJobsMap(gcInterval)

	// run job 1, round 1 - all entries marked
	runScript(t, jobsMap.get("job", "0"), job1Script1)
	// sleep longer than gcInterval to enable job gc in the next run
	time.Sleep(2 * gcInterval)
	// run job 2, round1 - trigger job gc, unmarking all entries
	runScript(t, jobsMap.get("job", "1"), job2Script1)
	// sleep longer than gcInterval to enable job gc in the next run
	time.Sleep(2 * gcInterval)
	// re-run job 2, round1 - trigger job gc, removing unmarked entries
	runScript(t, jobsMap.get("job", "1"), job2Script1)
	// ensure that at least one jobsMap.gc() completed
	jobsMap.gc()
	// run job 1, round 2 - verify that all job 1 timeseries have been gc'd
	runScript(t, jobsMap.get("job", "0"), job1Script2)
}

type metricsAdjusterTest struct {
	description string
	metrics     pmetric.MetricSlice
	adjusted    pmetric.MetricSlice
}

func runScript(t *testing.T, tsm *timeseriesMap, script []*metricsAdjusterTest) {
	l := zap.NewNop()
	t.Cleanup(func() { require.NoError(t, l.Sync()) }) // flushes buffer, if any
	ma := NewMetricsAdjuster(tsm, l)

	for _, test := range script {
		ma.AdjustMetricSlice(test.metrics)
		adjusted := test.metrics
		assert.EqualValuesf(t, test.adjusted, adjusted, "Test: %v - expected: %v, actual: %v", test.description, test.adjusted, adjusted)
	}
}
