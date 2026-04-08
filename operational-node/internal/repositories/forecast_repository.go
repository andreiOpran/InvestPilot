package repositories

import (
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type ForecastRepository interface {
	CreateForecast(taskID string, userID uint, forecast *models.ForecastResult) error
	GetForecast(taskID string, userID uint) (*models.ForecastResult, error)
}

type forecastRepository struct {
	db *gorm.DB
}

func NewForecastRepository(db *gorm.DB) ForecastRepository {
	return &forecastRepository{db: db}
}

func (r *forecastRepository) CreateForecast(taskID string, userID uint, forecast *models.ForecastResult) error {
	return r.db.Create(&forecast).Error
}

func (r *forecastRepository) GetForecast(taskID string, userID uint) (*models.ForecastResult, error) {
	var forecast models.ForecastResult
	err := r.db.Where("task_id  = ? AND user_id = ?", taskID, userID).First(&forecast).Error
	if err != nil {
		return nil, err
	}
	return &forecast, nil
}
