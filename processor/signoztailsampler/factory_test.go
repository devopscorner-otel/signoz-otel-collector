package signoztailsampler

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SigNoz/signoz-otel-collector/processor/signoztailsampler/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateProcessor(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "tail_sampling_config.yaml"))
	require.NoError(t, err)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(component.NewIDWithName(metadata.Type, "").String())
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(cfg))

	params := processortest.NewNopSettings(metadata.Type)
	tp, err := factory.CreateTraces(context.Background(), params, cfg, consumertest.NewNop())
	assert.NotNil(t, tp)
	assert.NoError(t, err, "cannot create trace processor")

	// this will cause the processor to properly initialize, so that we can later shutdown and
	// have all the go routines cleanly shut down
	assert.NoError(t, tp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tp.Shutdown(context.Background()))
}
