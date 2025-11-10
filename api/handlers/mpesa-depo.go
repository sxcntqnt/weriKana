// transfer.go (Daraja call orchestration + ledger record)
package cashdist

import (
    "context"
    "fmt"
    "time"
    "yourapp/models" // your GORM models package
)

// SendAllocationBatch sends out transfers and records transactions in DB
func SendAllocationBatch(ctx context.Context, allocs []AllocationResult, dryRun bool) error {
    for _, a := range allocs {
        if a.AmountToSend <= 0 { continue }
        // build idempotency key
        idempotency := fmt.Sprintf("%s-%d-%d", a.Bookie.Name, time.Now().Unix(), a.AmountToSend)

        // record pending transaction in DB (idempotency key)
        tx := models.Transaction{
            CustomerID: 0, // system sender id
            Amount: float64(a.AmountToSend),
            Type: "Deposit",
            Status: "Pending",
            Description: fmt.Sprintf("Auto-funding %s", a.Bookie.Name),
            RecipientReference: a.Bookie.MpesaNumber,
            IdempotencyKey: idempotency,
        }
        if err := models.DB.Create(&tx).Error; err != nil {
            return err
        }

        if dryRun {
            // just log, do not call Daraja
            fmt.Printf("[DRY] Would send KES %d to %s (%s)\n", a.AmountToSend, a.Bookie.Name, a.Bookie.MpesaNumber)
            tx.Status = "DryRun"
            models.DB.Save(&tx)
            continue
        }

        // Make Daraja STK push or B2C call
        mpesaResp, err := SendDarajaSTK(a.Bookie.MpesaNumber, a.AmountToSend, idempotency)
        if err != nil {
            // record failure & maybe schedule retry
            tx.Status = "Failed"
            tx.Meta = err.Error()
            models.DB.Save(&tx)
            // optionally continue with other transfers
            continue
        }
        // Update tx with mpesa ref
        tx.Status = "Initiated"
        tx.ThirdPartyRef = mpesaResp.TransactionID
        models.DB.Save(&tx)
    }
    return nil
}
