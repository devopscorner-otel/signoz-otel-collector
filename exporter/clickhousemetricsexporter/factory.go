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

package clickhousemetricsexporter

import (
	"context"
	"errors"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/SigNoz/signoz-otel-collector/exporter/clickhousemetricsexporter/internal/metadata"
)

// NewFactory creates a new Prometheus Remote Write exporter.
func NewFactory() exporter.Factory {

	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability))
}

func createMetricsExporter(ctx context.Context, set exporter.Settings,
	cfg component.Config) (exporter.Metrics, error) {

	prwCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid configuration")
	}

	prwe, err := NewPrwExporter(prwCfg, set)
	if err != nil {
		return nil, err
	}

	// Don't support the queue.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/2949.
	// Prometheus remote write samples needs to be in chronological
	// order for each timeseries. If we shard the incoming metrics
	// without considering this limitation, we experience
	// "out of order samples" errors.
	exporter, err := exporterhelper.NewMetrics(
		ctx,
		set,
		cfg,
		prwe.PushMetrics,
		exporterhelper.WithTimeout(prwCfg.TimeoutConfig),
		exporterhelper.WithQueue(exporterhelper.QueueBatchConfig{
			Enabled:      prwCfg.RemoteWriteQueue.Enabled,
			Sizer:        exporterhelper.RequestSizerTypeRequests,
			NumConsumers: 1,
			QueueSize:    int64(prwCfg.RemoteWriteQueue.QueueSize),
		}),
		exporterhelper.WithRetry(prwCfg.BackOffConfig),
		exporterhelper.WithStart(prwe.Start),
		exporterhelper.WithShutdown(prwe.Shutdown),
	)

	if err != nil {
		return nil, err
	}

	return resourcetotelemetry.WrapMetricsExporter(prwCfg.ResourceToTelemetrySettings, exporter), nil
}

func createDefaultConfig() component.Config {
	return &Config{
		Namespace:      "",
		ExternalLabels: map[string]string{},
		TimeoutConfig:  exporterhelper.NewDefaultTimeoutConfig(),
		BackOffConfig:  configretry.NewDefaultBackOffConfig(),
		HTTPClientSettings: confighttp.ClientConfig{
			Endpoint: "http://some.url:9411/api/prom/push",
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			ReadBufferSize:  0,
			WriteBufferSize: 512 * 1024,
			Timeout:         exporterhelper.NewDefaultTimeoutConfig().Timeout,
			Headers:         map[string]configopaque.String{},
		},
		// TODO(jbd): Adjust the default queue size.
		RemoteWriteQueue: RemoteWriteQueue{
			Enabled:      true,
			QueueSize:    10000,
			NumConsumers: 5,
		},
		WatcherInterval: 30 * time.Second,
		WriteTSToV4:     true,
		DisableV2:       false,
		EnableExpHist:   false,
	}
}
