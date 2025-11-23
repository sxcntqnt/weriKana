// db/transactions.go
package db

import (
	"gorm.io/gorm"

	"weriKana/models"
)

type TransactionsRepo interface {
	CreateTransaction(transaction *models.Transaction) error
	GetTransactionByReference(ref string) (*models.Transaction, error)
	ListTransactionsByPartner(partnerID string, limit int) ([]models.Transaction, error)
	UpdateTransactionStatus(id uint, status string) error
	// Add more methods as needed
}

type transactionsRepo struct {
	db *gorm.DB
}

// NewTransactionsRepo returns a new TransactionsRepo instance
func NewTransactionsRepo(db *gorm.DB) TransactionsRepo {
	return &transactionsRepo{db: db}
}

func (r *transactionsRepo) CreateTransaction(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *transactionsRepo) GetTransactionByReference(ref string) (*models.Transaction, error) {
	var txn models.Transaction
	err := r.db.Where("reference = ?", ref).First(&txn).Error
	if err != nil {
		return nil, err
	}
	return &txn, nil
}

func (r *transactionsRepo) ListTransactionsByPartner(partnerID string, limit int) ([]models.Transaction, error) {
	var txns []models.Transaction
	err := r.db.Where("partner_id = ?", partnerID).Limit(limit).Find(&txns).Error
	if err != nil {
		return nil, err
	}
	return txns, nil
}

func (r *transactionsRepo) UpdateTransactionStatus(id uint, status string) error {
	return r.db.Model(&models.Transaction{}).Where("id = ?", id).Update("status", status).Error
}
