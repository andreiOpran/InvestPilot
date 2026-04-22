package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type TransactionService interface {
	GetTransactionHistory(userID uint, page, limit int) (*models.PaginatedTransactionsResponse, error)
}

type transactionService struct {
	repo repositories.TransactionRepository
}

func NewTransactionService(repo repositories.TransactionRepository) TransactionService {
	return &transactionService{repo: repo}
}

func (s *transactionService) GetTransactionHistory(userID uint, page, limit int) (*models.PaginatedTransactionsResponse, error) {
	// apply defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > config.Env.TransactionCountLimit {
		limit = config.Env.TransactionCountDefault
	}

	offset := (page - 1) * limit

	data, total, err := s.repo.GetUnifiedHistory(userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// avoid returnin nil json arrays
	if data == nil {
		data = []models.UnifiedTransaction{}
	}

	return &models.PaginatedTransactionsResponse{
		Data:       data,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
	}, nil
}
