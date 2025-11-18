// api/handlers/allocation.go
package handlers

import (
    "math"
    "github.com/google/uuid"
    "weriKana/models"
)

func AllocateFunds(totalBalance int64, reservePct float64, bookies []models.Bookie, beta float64, minSend int64) []models.AllocationResult {
    reserve := int64(float64(totalBalance) * reservePct)
    allocatable := totalBalance - reserve
    if allocatable <= 0 {
        return nil
    }

    n := len(bookies)
    perf := make([]float64, n)
    risk := make([]float64, n)

    for i, b := range bookies {
        perf[i] = math.Max(0, b.RecentLogRet)
        vol := math.Max(b.RecentVol, 1e-6)
        risk[i] = 1.0 / vol
    }

    norm := func(arr []float64) []float64 {
        var sum float64
        for _, v := range arr { sum += v }
        if sum == 0 {
            res := make([]float64, n)
            for i := range res { res[i] = 1.0 / float64(n) }
            return res
        }
        res := make([]float64, n)
        for i, v := range arr { res[i] = v / sum }
        return res
    }

    pweights := norm(perf)
    rweights := norm(risk)

    scores := make([]float64, n)
    for i := range bookies {
        scores[i] = beta*pweights[i] + (1-beta)*rweights[i]
    }
    scores = norm(scores)

    results := make([]models.AllocationResult, 0, n)
    remaining := allocatable

    for i, b := range bookies {
        amt := int64(math.Floor(float64(allocatable) * scores[i]))

        if amt > 0 && amt < b.MinDeposit {
            if remaining >= b.MinDeposit {
                amt = b.MinDeposit
            } else {
                amt = 0
            }
        }
        if amt > b.MaxDeposit {
            amt = b.MaxDeposit
        }
        if amt < minSend {
            amt = 0
        }

        remaining -= amt
        if amt > 0 {
            results = append(results, models.AllocationResult{
                BookieID:       b.ID,
                BookieName:     b.Name,
                MpesaNumber:    b.MpesaNumber,
                AmountToSend:   amt,
                Proportion:     scores[i],
                IsReal:         true,
                IdempotencyKey: uuid.New().String(),
                TransactionID:  uuid.New(),
            })
        }
    }

    // Distribute any remaining amount
    if remaining > 0 {
        for remaining > 0 {
            bestIdx := -1
            bestScore := -1.0
            for i, s := range scores {
                if i >= len(results) { continue }
                if s > bestScore && results[i].AmountToSend < bookies[i].MaxDeposit {
                    bestScore = s
                    bestIdx = i
                }
            }
            if bestIdx == -1 { break }
            add := int64(math.Min(float64(remaining), float64(bookies[bestIdx].MaxDeposit - results[bestIdx].AmountToSend)))
            if add <= 0 { break }
            results[bestIdx].AmountToSend += add
            remaining -= add
        }
    }

    return results
}
