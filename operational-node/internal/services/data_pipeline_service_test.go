package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/clients"
)

func TestRunDailyPipeline_nilPublisher_returnsNil(t *testing.T) {
	svc := NewDataPipelineService()

	// publisher is nil -> service returns nil immediately (no RabbitMQ required)
	prev := clients.Publisher
	clients.Publisher = nil
	defer func() { clients.Publisher = prev }()

	err := svc.RunDailyPipeline()
	assert.NoError(t, err)
}

func TestRunIntradayPipeline_nilPublisher_returnsNil(t *testing.T) {
	svc := NewDataPipelineService()

	prev := clients.Publisher
	clients.Publisher = nil
	defer func() { clients.Publisher = prev }()

	err := svc.RunIntradayPipeline()
	assert.NoError(t, err)
}
