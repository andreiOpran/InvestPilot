package repositories

import (
	"time"

	"gorm.io/gorm"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

// CleanupRepository encapsulates the logic for garbage collection cron jobs
type CleanupRepository interface {
	DeleteExpiredActionTokens(now time.Time) (int64, error)
	DeleteExpiredSessionsBatch(now time.Time, batchSize int) (int64, error)
}

type cleanupRepository struct {
	db *gorm.DB
}

func NewCleanupRepository(db *gorm.DB) CleanupRepository {
	return &cleanupRepository{db: db}
}

func (r *cleanupRepository) DeleteExpiredActionTokens(now time.Time) (int64, error) {
	res := r.db.Where("expires_at < ?", now).Delete(&models.ActionToken{})
	return res.RowsAffected, res.Error
}

func (r *cleanupRepository) DeleteExpiredSessionsBatch(now time.Time, batchSize int) (int64, error) {
	// subquery to safely chunk deletions without locking the entire table
	subQuery := r.db.Table("sessions").Select("id").Where("expires_at < ?", now).Limit(batchSize)
	res := r.db.Where("id IN (?)", subQuery).Delete(&models.Session{})
	return res.RowsAffected, res.Error
}
