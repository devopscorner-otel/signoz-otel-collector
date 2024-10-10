// Brought in as is from opentelemetry-collector-contrib

package trace

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}