package bank

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"weriKana/models"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// =====================================================================
// CONFIG
// =====================================================================
const (
	cacheTTL         = 5 * time.Minute
	customerCacheKey = "balance:customer:%s"

	warmupWorkers   = 32
	warmupBatchSize = 1000
	warmupInterval  = 20 * time.Second
)

// =====================================================================
// PUBLIC TYPES
// =====================================================================
type AccountType string

const (
	Sports AccountType = "sports"
	Stocks AccountType = "stocks"
	Forex  AccountType = "forex"
	Crypto AccountType = "crypto"
)

type BalanceSummary struct {
	CustomerID   uuid.UUID         `json:"customer_id"`
	CustomerName string            `json:"customer_name"`
	TotalReal    int64             `json:"total_real_cents"`
	TotalFake    int64             `json:"total_fake_cents"`
	ByVertical   []VerticalBalance `json:"by_vertical"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type VerticalBalance struct {
	Vertical     string `json:"vertical"`
	BankName     string `json:"bank_name"`
	RealCents    int64  `json:"real_cents"`
	FakeCents    int64  `json:"fake_cents"`
	AccountCount int    `json:"account_count"`
}

// =====================================================================
// BALANCE ENGINE — FINAL, FLAWLESS
// =====================================================================
type BalanceEngine struct {
	db    *gorm.DB
	redis *redis.Client
	sf    singleflight.Group

	shutdown         chan struct{}
	cacheWriteCtx    context.Context
	cacheWriteCancel context.CancelFunc

	// Metrics
	cacheHit      uint64
	cacheMiss     uint64
	dbFallback    uint64
	warmupSuccess uint64
	warmupFailed  uint64

	wg sync.WaitGroup
}

func NewBalanceEngine(db *gorm.DB, redisClient *redis.Client) *BalanceEngine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &BalanceEngine{
		db:               db,
		redis:            redisClient,
		shutdown:         make(chan struct{}),
		cacheWriteCtx:    ctx,
		cacheWriteCancel: cancel,
	}
	e.wg.Add(1)
	go e.runWarmupController()
	return e
}

// =====================================================================
// PUBLIC: GET BALANCE
// =====================================================================
func (e *BalanceEngine) GetSharpBalance(ctx context.Context, customerID uuid.UUID) (*BalanceSummary, error) {
	key := fmt.Sprintf(customerCacheKey, customerID)

	if data, err := e.redis.Get(ctx, key).Bytes(); err == nil {
		var summary BalanceSummary
		if err := json.Unmarshal(data, &summary); err == nil {
			atomic.AddUint64(&e.cacheHit, 1)
			return &summary, nil
		}
		slog.Warn("balance.cache_corrupted", "customer_id", customerID, "err", err)
		_ = e.redis.Del(ctx, key).Err()
	} else if err != redis.Nil {
		slog.Error("balance.redis_connection_failed", "err", err)
	}

	v, err, _ := e.sf.Do(key, func() (interface{}, error) {
		atomic.AddUint64(&e.dbFallback, 1)
		return e.computeFromDB(ctx, customerID)
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("customer_not_found: %w", err)
		}
		return nil, fmt.Errorf("balance_compute_failed: %w", err)
	}

	summary := v.(*BalanceSummary)
	atomic.AddUint64(&e.cacheMiss, 1)
	go e.writeCacheAsync(customerID, summary)
	return summary, nil
}

func (e *BalanceEngine) writeCacheAsync(customerID uuid.UUID, summary *BalanceSummary) {
	select {
	case <-e.cacheWriteCtx.Done():
		return
	default:
	}

	data, err := json.Marshal(summary)
	if err != nil {
		slog.Error("balance.marshal_failed", "customer_id", customerID, "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(e.cacheWriteCtx, 3*time.Second)
	defer cancel()

	key := fmt.Sprintf(customerCacheKey, customerID)
	if err := e.redis.Set(ctx, key, data, cacheTTL).Err(); err != nil {
		slog.Error("balance.cache_write_failed", "customer_id", customerID, "err", err)
	}
}

// =====================================================================
// PUBLIC: ATOMIC UPDATE + LEDGER
// =====================================================================
type balanceUpdater interface {
	AddReal(int64)
	AddFake(int64)
}

// Updated BalanceEngine method — Aligned with new TransactionLog: SharpID, RealCents/FakeCents/BonusCents.
// Optional TransactionID param for linking (set if from Transaction; else uuid.New()).

func (e *BalanceEngine) UpdateBalanceAtomic(
	ctx context.Context,
	sharpID uuid.UUID, // Changed: SharpID replaces customerID
	accountType AccountType,
	accountID uuid.UUID,
	deltaReal, deltaFake int64,
	reason, ref string,
	transactionID ...uuid.UUID, // Optional: Link to source Transaction
) error {
	var txID uuid.UUID
	if len(transactionID) > 0 {
		txID = transactionID[0]
	} else {
		txID = uuid.New() // Generate if not provided
	}

	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var updater balanceUpdater
		switch accountType {
		case Sports:
			var acc models.SportsAccount
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&acc, "id = ?", accountID).Error; err != nil {
				return err
			}
			updater = &acc
		case Stocks:
			var acc models.StockAccount
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&acc, "id = ?", accountID).Error; err != nil {
				return err
			}
			updater = &acc
		case Forex:
			var acc models.ForexAccount
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&acc, "id = ?", accountID).Error; err != nil {
				return err
			}
			updater = &acc
		case Crypto:
			var acc models.CryptoAccount
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&acc, "id = ?", accountID).Error; err != nil {
				return err
			}
			updater = &acc
		default:
			return fmt.Errorf("unsupported account type: %s", accountType)
		}
		updater.AddReal(deltaReal)
		updater.AddFake(deltaFake)
		if err := tx.Save(updater).Error; err != nil {
			return err
		}
		ledger := models.TransactionLog{
			ID:          uuid.New(),
			SharpID:     sharpID, // Updated: SharpID replaces CustomerID
			AccountType: string(accountType),
			AccountID:   accountID,
			RealCents:   deltaReal, // Maps to RealCents
			FakeCents:   deltaFake, // Maps to FakeCents (bonus/simulated)
			BonusCents:  deltaFake, // If bonus subset of fake; adjust if separate
			FiatCentsKE: 0,         // Default 0; set if crypto
			Reference:   ref,
			Reason:      reason,
			Status:      "COMPLETED",
			CreatedAt:   time.Now().UTC(),
			TransactionID: txID, // Link back to source Transaction (optional)
		}
		return tx.Create(&ledger).Error
	})
	if err == nil {
		e.invalidateCustomerCache(sharpID) // Updated: Use SharpID for cache key
	}
	return err
}
func (e *BalanceEngine) invalidateCustomerCache(customerID uuid.UUID) {
	key := fmt.Sprintf(customerCacheKey, customerID)
	ctx, cancel := context.WithTimeout(e.cacheWriteCtx, 2*time.Second)
	defer cancel()
	_ = e.redis.Del(ctx, key).Err()
	e.sf.Forget(key)
}

// =====================================================================
// ADMIN: CURSOR PAGINATION
// =====================================================================
// Updated GetAllBalancesPaginated — Aligned with Sharp merge: Queries Sharp IDs; uses GetSharpBalance.
// Paginate summaries by Sharp ID cursor for admin overviews.

func (e *BalanceEngine) GetAllBalancesPaginated(ctx context.Context, limit int, cursor string) ([]BalanceSummary, string, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	var ids []uuid.UUID
	q := e.db.WithContext(ctx).Model(&models.Sharp{}).Order("id ASC") // Updated: Model(&models.Sharp{})
	if cursor != "" {
		if curID, err := uuid.Parse(cursor); err == nil {
			q = q.Where("id > ?", curID)
		}
	}
	if err := q.Limit(limit+1).Pluck("id", &ids).Error; err != nil {
		return nil, "", err
	}
	summaries := make([]BalanceSummary, 0, limit)
	var nextCursor string
	for i, id := range ids {
		if i == limit {
			nextCursor = id.String()
			break
		}
		if bal, err := e.GetSharpBalance(ctx, id); err == nil { // Updated: GetSharpBalance replaces GetCustomerBalance
			summaries = append(summaries, *bal)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("admin.balance_load_failed", "sharp_id", id, "err", err) // Updated: sharp_id
		}
	}
	return summaries, nextCursor, nil
}

// =====================================================================
// BACKGROUND WARMUP — SINGLE QUERY, NO DOUBLE HIT
// =====================================================================
func (e *BalanceEngine) runWarmupController() {
	defer e.wg.Done()
	ticker := time.NewTicker(warmupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.shutdown:
			return
		case <-ticker.C:
			e.warmupWithNumericID()
		}
	}
}

func (e *BalanceEngine) warmupWithNumericID() {
	type sharpIDPair struct {
		ID       uuid.UUID `gorm:"column:id"`
		NumericID int64    `gorm:"column:numeric_id"`
	}
	var lastNumericID int64
	batchCh := make(chan []sharpIDPair, warmupWorkers)
	go func() {
		defer close(batchCh)
		for {
			var batch []sharpIDPair
			q := e.db.Model(&models.Sharp{}). // Updated: Model(&models.Sharp{})
				Select("id, numeric_id").
				Order("numeric_id")
			if lastNumericID > 0 {
				q = q.Where("numeric_id > ?", lastNumericID)
			}
			if err := q.Limit(warmupBatchSize).Scan(&batch).Error; err != nil {
				slog.Error("warmup.scan_failed", "err", err)
				atomic.AddUint64(&e.warmupFailed, 1)
				return
			}
			if len(batch) == 0 {
				return
			}
			lastNumericID = batch[len(batch)-1].NumericID
			atomic.AddUint64(&e.warmupSuccess, uint64(len(batch)))
			select {
			case batchCh <- batch:
			case <-e.shutdown:
				return
			}
		}
	}()
	var wg sync.WaitGroup
	for i := 0; i < warmupWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range batchCh {
				for _, pair := range batch {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					_, _ = e.GetSharpBalance(ctx, pair.ID) 
					cancel()
				}
			}
		}()
	}
	wg.Wait()
}
// =====================================================================
// PRIVATE: DB COMPUTE + AGGREGATION
// =====================================================================
func (e *BalanceEngine) computeFromDB(ctx context.Context, sharpID uuid.UUID) (*BalanceSummary, error) { // Updated: sharpID replaces customerID
	var nexus models.AssetNexus
	err := e.db.WithContext(ctx).
		Preload("Sharp"). // Updated: Preload("Sharp") replaces "Customer"
		Preload("SportsAccounts", "deleted_at IS NULL").
		Preload("StockAccounts", "deleted_at IS NULL").
		Preload("ForexAccounts", "deleted_at IS NULL").
		Preload("CryptoAccounts", "deleted_at IS NULL").
		First(&nexus, "sharp_id = ?", sharpID).Error // Updated: "sharp_id = ?" replaces "customer_id = ?"
	if err != nil {
		return nil, err
	}
	summary := &BalanceSummary{
		CustomerID:    sharpID, // Updated: Use sharpID (or rename field to SharpID in summary)
		CustomerName:  nexus.Sharp.Name, // Updated: nexus.Sharp.Name replaces nexus.Customer.Name
		UpdatedAt:     time.Now().UTC(),
		ByVertical:    make([]VerticalBalance, 0, 4),
	}
	e.aggregateVertical(summary, "Sports", nexus.SportsAccounts)
	e.aggregateVertical(summary, "Stocks", nexus.StockAccounts)
	e.aggregateVertical(summary, "Forex", nexus.ForexAccounts)
	e.aggregateVertical(summary, "Crypto", nexus.CryptoAccounts)
	return summary, nil
}

func (e *BalanceEngine) aggregateVertical(summary *BalanceSummary, name string, accounts any) {
	var real, fake int64
	count := 0

	switch accs := accounts.(type) {
	case []models.SportsAccount:
		for _, a := range accs {
			real += a.RealBalanceCents
			fake += a.FakeBalanceCents
			count++
		}
	case []models.StockAccount:
		for _, a := range accs {
			real += a.RealBalanceCents
			fake += a.FakeBalanceCents
			count++
		}
	case []models.ForexAccount:
		for _, a := range accs {
			real += a.RealBalanceCents
			fake += a.FakeBalanceCents
			count++
		}
	case []models.CryptoAccount:
		for _, a := range accs {
			real += a.RealBalanceCents
			fake += a.FakeBalanceCents
			count++
		}
	}

	if count > 0 {
		summary.ByVertical = append(summary.ByVertical, VerticalBalance{
			Vertical:     name,
			BankName:     fmt.Sprintf("Sharps %s Bank", name),
			RealCents:    real,
			FakeCents:    fake,
			AccountCount: count,
		})
		summary.TotalReal += real
		summary.TotalFake += fake
	}
}

// =====================================================================
// METRICS & SHUTDOWN
// =====================================================================
func (e *BalanceEngine) Metrics() map[string]uint64 {
	return map[string]uint64{
		"cache_hit":      atomic.LoadUint64(&e.cacheHit),
		"cache_miss":     atomic.LoadUint64(&e.cacheMiss),
		"db_fallback":    atomic.LoadUint64(&e.dbFallback),
		"warmup_success": atomic.LoadUint64(&e.warmupSuccess),
		"warmup_failed":  atomic.LoadUint64(&e.warmupFailed),
	}
}

func (e *BalanceEngine) Shutdown() {
	e.cacheWriteCancel()
	close(e.shutdown)
	e.wg.Wait()
}
