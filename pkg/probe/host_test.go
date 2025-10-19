package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHostCollectorWithPath(t *testing.T) {
	customPath := "/custom/proc/path"
	collector := NewHostCollectorWithPath(customPath)

	assert.NotNil(t, collector)
	assert.Equal(t, customPath, collector.procPath)
}

func TestNewHostCollector(t *testing.T) {
	collector := NewHostCollector()

	assert.NotNil(t, collector)
	assert.Equal(t, "/proc", collector.procPath)
}
