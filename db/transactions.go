package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
        "gorm.io/gorm"

)

type TransactionsRepo interface {
}

type transactionsRepo struct {
	db *gorm.DB
}

func NewTransactionsRepo(db *gorm.DB) *transactionsRepo {
	return &transactionsRepo{db: db}
}
