package servicemocks

import "github.com/stretchr/testify/mock"

type MockDataPipelineService struct {
	mock.Mock
}

func (m *MockDataPipelineService) RunDailyPipeline() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDataPipelineService) RunIntradayPipeline() error {
	args := m.Called()
	return args.Error(0)
}
