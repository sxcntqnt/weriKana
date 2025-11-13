package db

import (
    "gorm.io/gorm"
)

type TransactionsRepo interface {
    // Define repository methods here, for example:
    // CreateTransaction(transaction *models.Transaction) error
}

type transactionsRepo struct {
    db *gorm.DB
}

// NewTransactionsRepo returns a new TransactionsRepo instance
func NewTransactionsRepo(db *gorm.DB) *transactionsRepo {
    return &transactionsRepo{db: db}
}

