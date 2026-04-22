package repositories

import (
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type TransactionRepository interface {
	GetUnifiedHistory(userID uint, limit, offset int) ([]models.UnifiedTransaction, int64, error)
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) GetUnifiedHistory(userID uint, limit, offset int) ([]models.UnifiedTransaction, int64, error) {
	var totalCount int64

	// get total combined count for pagination
	// define one subquery for each table
	countQ1 := r.db.Table("fundings").Select("id").Where("user_id = ?", userID)
	countQ2 := r.db.Table("transactions").Select("id").Where("user_id = ?", userID)

	// union the count subqueries into `totalCount`
	if err := r.db.Table("(?) as combined", r.db.Raw("? UNION ALL ?", countQ1, countQ2)).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// fetch actual unified data, injecting missing status for `transactions`
	// define one subquery for each table
	dataQ1 := r.db.Table("fundings").
		Select("id, 'FUNDING' as source, type, amount, status, created_at as timestamp").
		Where("user_id = ?", userID)

	dataQ2 := r.db.Table("transactions").
		Select("id, 'TRANSACTION' as source, type, amount, 'COMPLETED' as status, created_at as timestamp").
		Where("user_id = ?", userID)

	// union the data subqueries into `results`
	var results []models.UnifiedTransaction
	err := r.db.Table("(?) as combined", r.db.Raw("? UNION ALL ?", dataQ1, dataQ2)).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Scan(&results).Error

	if err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}
