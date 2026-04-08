package repositories

import (
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type ForecastRepository interface {
	CreatePendingForecast(taskID string) error
	GetForecastByTaskID(taskID string) (*models.ForecastResult, error)
}

type forecastRepository struct {
	db *gorm.DB
}

func NewForecastRepository(db *gorm.DB) ForecastRepository {
	return &forecastRepository{db: db}
}

func (r *forecastRepository) CreatePendingForecast(taskID string) error {
	forecast := models.ForecastResult{
		TaskID: taskID,
		Status: "pending",
	}
	return r.db.Create(&forecast).Error
}

func (r *forecastRepository) GetForecastByTaskID(taskID string) (*models.ForecastResult, error) {
	var forecast models.ForecastResult
	err := r.db.Where("task_id  = ?", taskID).First(&forecast).Error
	if err != nil {
		return nil, err
	}
	return &forecast, nil
}
