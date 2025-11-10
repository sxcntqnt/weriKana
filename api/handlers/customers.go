package handlers

import (
	"net/http"

	"github.com/goremit/money-transfer/db"
)
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Account struct {
	UserID     int     `json:"user_id"`
	BookieName string  `json:"bookie_name"`
	RealBal    float64 `json:"real_balance"` // Encrypted in DB
	FakeBal    float64 `json:"fake_balance"`
}

type Customers struct {
	transactionsRepo db.TransactionsRepo
}

func NewCustomers(t db.TransactionsRepo) *Customers {
	return &Customers{transactionsRepo: t}
}

func (c *Customers) HandleShowTransaction(w http.ResponseWriter, r *http.Request) {
	//
}

func (c *Customers) HandleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	//
}

func (c *Customers) HandleFundTransaction(w http.ResponseWriter, r *http.Request) {
	//
}

func (c *Customers) HandleCancelTransaction(w http.ResponseWriter, r *http.Request) {
	//
}
