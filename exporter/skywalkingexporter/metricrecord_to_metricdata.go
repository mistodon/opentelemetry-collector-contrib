// Copyright 2020, OpenTelemetry Authors
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

package skywalkingexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/skywalkingexporter"

import (
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	metricpb "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

const (
	defaultServiceInstance = "otel-collector-instance"
)

func resourceToMetricLabels(resource pcommon.Resource) []*metricpb.Label {
	attrs := resource.Attributes()
	labels := make([]*metricpb.Label, 0, attrs.Len())
	attrs.Range(func(k string, v pcommon.Value) bool {
		labels = append(labels,
			&metricpb.Label{
				Name:  k,
				Value: v.AsString(),
			})
		return true
	})
	return labels
}

func resourceToServiceInfo(resource pcommon.Resource) (service string, serviceInstance string) {
	attrs := resource.Attributes()
	if serviceName, ok := attrs.Get(conventions.AttributeServiceName); ok {
		service = serviceName.AsString()
	} else {
		service = defaultServiceName
	}
	if serviceInstanceID, ok := attrs.Get(conventions.AttributeServiceInstanceID); ok {
		serviceInstance = serviceInstanceID.AsString()
	} else {
		serviceInstance = defaultServiceInstance
	}
	return service, serviceInstance
}

func numberMetricsToData(name string, data pmetric.NumberDataPointSlice, defaultLabels []*metricpb.Label) (metrics []*metricpb.MeterData) {
	metrics = make([]*metricpb.MeterData, 0, data.Len())
	for i := 0; i < data.Len(); i++ {
		dataPoint := data.At(i)
		attributeMap := dataPoint.Attributes()
		labels := make([]*metricpb.Label, 0, attributeMap.Len()+len(defaultLabels))
		attributeMap.Range(func(k string, v pcommon.Value) bool {
			labels = append(labels, &metricpb.Label{Name: k, Value: v.AsString()})
			return true
		})

		for _, label := range defaultLabels {
			labels = append(labels, &metricpb.Label{Name: label.Name, Value: label.Value})
		}
		meterData := &metricpb.MeterData{}
		sv := &metricpb.MeterData_SingleValue{SingleValue: &metricpb.MeterSingleValue{}}
		sv.SingleValue.Labels = labels
		meterData.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
		sv.SingleValue.Name = name
		switch dataPoint.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			sv.SingleValue.Value = float64(dataPoint.IntVal())
		case pmetric.NumberDataPointValueTypeDouble:
			sv.SingleValue.Value = dataPoint.DoubleVal()
		}
		meterData.Metric = sv
		metrics = append(metrics, meterData)
	}
	return metrics
}

func doubleHistogramMetricsToData(name string, data pmetric.HistogramDataPointSlice, defaultLabels []*metricpb.Label) (metrics []*metricpb.MeterData) {
	metrics = make([]*metricpb.MeterData, 0, 3*data.Len())
	for i := 0; i < data.Len(); i++ {
		dataPoint := data.At(i)
		attributeMap := dataPoint.Attributes()
		labels := make([]*metricpb.Label, 0, attributeMap.Len()+len(defaultLabels))
		attributeMap.Range(func(k string, v pcommon.Value) bool {
			labels = append(labels, &metricpb.Label{Name: k, Value: v.AsString()})
			return true
		})

		for _, label := range defaultLabels {
			labels = append(labels, &metricpb.Label{Name: label.Name, Value: label.Value})
		}

		meterData := &metricpb.MeterData{}
		hg := &metricpb.MeterData_Histogram{Histogram: &metricpb.MeterHistogram{}}
		hg.Histogram.Labels = labels
		hg.Histogram.Name = name
		bounds := dataPoint.ExplicitBounds()
		bucketCount := dataPoint.BucketCounts().Len()

		if bucketCount > 0 {
			hg.Histogram.Values = append(hg.Histogram.Values,
				&metricpb.MeterBucketValue{Count: int64(dataPoint.BucketCounts().At(0)), IsNegativeInfinity: true})
		}
		for i := 1; i < bucketCount && i-1 < bounds.Len(); i++ {
			hg.Histogram.Values = append(hg.Histogram.Values, &metricpb.MeterBucketValue{Bucket: bounds.At(i - 1),
				Count: int64(dataPoint.BucketCounts().At(i))})
		}

		meterData.Metric = hg
		meterData.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
		metrics = append(metrics, meterData)

		meterDataSum := &metricpb.MeterData{}
		svs := &metricpb.MeterData_SingleValue{SingleValue: &metricpb.MeterSingleValue{}}
		svs.SingleValue.Labels = labels
		svs.SingleValue.Name = name + "_sum"
		svs.SingleValue.Value = dataPoint.Sum()
		meterDataSum.Metric = svs
		meterDataSum.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
		metrics = append(metrics, meterDataSum)

		meterDataCount := &metricpb.MeterData{}
		svc := &metricpb.MeterData_SingleValue{SingleValue: &metricpb.MeterSingleValue{}}
		svc.SingleValue.Labels = labels
		meterDataCount.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
		svc.SingleValue.Name = name + "_count"
		svc.SingleValue.Value = float64(dataPoint.Count())
		meterDataCount.Metric = svc
		metrics = append(metrics, meterDataCount)
	}
	return metrics
}

func doubleSummaryMetricsToData(name string, data pmetric.SummaryDataPointSlice, defaultLabels []*metricpb.Label) (metrics []*metricpb.MeterData) {
	metrics = make([]*metricpb.MeterData, 0, 3*data.Len())
	for i := 0; i < data.Len(); i++ {
		dataPoint := data.At(i)
		attributeMap := dataPoint.Attributes()
		labels := make([]*metricpb.Label, 0, attributeMap.Len()+len(defaultLabels))
		attributeMap.Range(func(k string, v pcommon.Value) bool {
			labels = append(labels, &metricpb.Label{Name: k, Value: v.AsString()})
			return true
		})

		for _, label := range defaultLabels {
			labels = append(labels, &metricpb.Label{Name: label.Name, Value: label.Value})
		}

		values := dataPoint.QuantileValues()
		for i := 0; i < values.Len(); i++ {
			value := values.At(i)
			meterData := &metricpb.MeterData{}
			sv := &metricpb.MeterData_SingleValue{SingleValue: &metricpb.MeterSingleValue{}}
			svLabels := make([]*metricpb.Label, 0, len(labels)+1)
			svLabels = append(svLabels, labels...)
			svLabels = append(svLabels, &metricpb.Label{Name: "quantile", Value: strconv.FormatFloat(value.Quantile(), 'g', -1, 64)})
			sv.SingleValue.Labels = svLabels
			meterData.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
			sv.SingleValue.Name = name
			sv.SingleValue.Value = value.Value()
			meterData.Metric = sv
			metrics = append(metrics, meterData)
		}

		meterDataSum := &metricpb.MeterData{}
		svs := &metricpb.MeterData_SingleValue{SingleValue: &metricpb.MeterSingleValue{}}
		svs.SingleValue.Labels = labels
		svs.SingleValue.Name = name + "_sum"
		svs.SingleValue.Value = dataPoint.Sum()
		meterDataSum.Metric = svs
		meterDataSum.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
		metrics = append(metrics, meterDataSum)

		meterDataCount := &metricpb.MeterData{}
		svc := &metricpb.MeterData_SingleValue{SingleValue: &metricpb.MeterSingleValue{}}
		svc.SingleValue.Labels = labels
		meterDataCount.Timestamp = dataPoint.Timestamp().AsTime().UnixMilli()
		svc.SingleValue.Name = name + "_count"
		svc.SingleValue.Value = float64(dataPoint.Count())
		meterDataCount.Metric = svc
		metrics = append(metrics, meterDataCount)
	}
	return metrics
}

func metricDataToSwMetricData(md pmetric.Metric, defaultLabels []*metricpb.Label) (metrics []*metricpb.MeterData) {
	switch md.DataType() {
	case pmetric.MetricDataTypeNone:
		break
	case pmetric.MetricDataTypeGauge:
		return numberMetricsToData(md.Name(), md.Gauge().DataPoints(), defaultLabels)
	case pmetric.MetricDataTypeSum:
		return numberMetricsToData(md.Name(), md.Sum().DataPoints(), defaultLabels)
	case pmetric.MetricDataTypeHistogram:
		return doubleHistogramMetricsToData(md.Name(), md.Histogram().DataPoints(), defaultLabels)
	case pmetric.MetricDataTypeSummary:
		return doubleSummaryMetricsToData(md.Name(), md.Summary().DataPoints(), defaultLabels)
	}
	return nil
}

func metricsRecordToMetricData(
	md pmetric.Metrics,
) (metrics *metricpb.MeterDataCollection) {
	resMetrics := md.ResourceMetrics()
	for i := 0; i < resMetrics.Len(); i++ {
		resMetricSlice := resMetrics.At(i)
		labels := resourceToMetricLabels(resMetricSlice.Resource())
		service, serviceInstance := resourceToServiceInfo(resMetricSlice.Resource())
		insMetricSlice := resMetricSlice.ScopeMetrics()
		metrics = &metricpb.MeterDataCollection{}
		metrics.MeterData = make([]*metricpb.MeterData, 0)
		for j := 0; j < insMetricSlice.Len(); j++ {
			insMetrics := insMetricSlice.At(j)
			// ignore insMetrics.Scope()
			metricSlice := insMetrics.Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				oneMetric := metricSlice.At(k)
				ms := metricDataToSwMetricData(oneMetric, labels)
				if ms == nil {
					continue
				}
				for _, m := range ms {
					m.Service = service
					m.ServiceInstance = serviceInstance
				}
				metrics.MeterData = append(metrics.MeterData, ms...)
			}
		}
	}
	return metrics
}
