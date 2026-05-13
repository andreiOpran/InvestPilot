package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestForecastRepository_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewForecastRepository(db)

	forecast := &models.ForecastResult{
		TaskID:  "task-test-001",
		UserID:  1,
		Status:  "pending",
		Payload: "",
	}

	err := repo.CreateForecast("task-test-001", 1, forecast)
	assert.NoError(t, err)
	assert.NotZero(t, forecast.ID)

	result, err := repo.GetForecast("task-test-001", 1)
	assert.NoError(t, err)
	assert.Equal(t, "task-test-001", result.TaskID)
	assert.Equal(t, uint(1), result.UserID)
	assert.Equal(t, "pending", result.Status)
}

func TestForecastRepository_GetForecast_wrongUser_returnsError(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewForecastRepository(db)

	forecast := &models.ForecastResult{
		TaskID: "task-user-mismatch",
		UserID: 2,
		Status: "complete",
	}
	_ = repo.CreateForecast("task-user-mismatch", 2, forecast)

	result, err := repo.GetForecast("task-user-mismatch", 99)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestForecastRepository_GetForecast_notFound_returnsError(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewForecastRepository(db)

	result, err := repo.GetForecast("does-not-exist", 1)
	assert.Error(t, err)
	assert.Nil(t, result)
}
