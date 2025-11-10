// allocation.go (core allocation + batch transfer logic)
package cashdist

import (
    "context"
    "fmt"
    "math"
    "time"
)

// Bookie struct simplified for allocation
type Bookie struct {
    Name           string
    MpesaNumber    string
    MinDeposit     int64 // KES in cents (or smallest unit you use)
    MaxDeposit     int64
    RecentLogRet   float64 // EWMA of log returns
    RecentVol      float64 // EWMA volatility
    CurrentBalance int64
}

// AllocationResult for each bookie
type AllocationResult struct {
    Bookie      Bookie
    AmountToSend int64
    Reason       string
}

// AllocateFunds computes the planned transfers
func AllocateFunds(totalBalance int64, reservePct float64, bookies []Bookie, beta float64, minSend int64) []AllocationResult {
    // 1. Reserve buffer
    reserve := int64(float64(totalBalance) * reservePct)
    allocatable := totalBalance - reserve
    if allocatable <= 0 { return nil } 

    // 2. Build two weight arrays: perf and risk
    n := len(bookies)
    perf := make([]float64, n)
    risk := make([]float64, n)
    for i, b := range bookies {
        // Perf weight: positive part of EWMA log return
        perf[i] = math.Max(0, b.RecentLogRet)
        // Risk weight: inverse vol (avoid division by zero)
        vol := math.Max(b.RecentVol, 1e-6)
        risk[i] = 1.0 / vol
    }

    // 3. Normalize helper
    norm := func(arr []float64) []float64 {
        sum := 0.0
        for _, v := range arr { sum += v }
        res := make([]float64, len(arr))
        if sum == 0 {
            // fallback equal weights
            for i := range arr { res[i] = 1.0 / float64(len(arr)) }
            return res
        }
        for i, v := range arr { res[i] = v / sum }
        return res
    }

    pweights := norm(perf)
    rweights := norm(risk)

    // 4. Combined score
    scores := make([]float64, n)
    for i := range bookies {
        scores[i] = beta*pweights[i] + (1.0-beta)*rweights[i]
    }
    scores = norm(scores)

    // 5. Map scores -> amounts (apply min constraints)
    results := make([]AllocationResult, 0, n)
    remaining := allocatable
    for i, b := range bookies {
        amt := int64(math.Floor(float64(allocatable) * scores[i]))
        // enforce min deposit
        if amt > 0 && amt < b.MinDeposit { 
            // if below min, round up to min if funds permit, otherwise zero
            if remaining >= b.MinDeposit {
                amt = b.MinDeposit
            } else {
                amt = 0
            }
        }
        if amt > b.MaxDeposit { amt = b.MaxDeposit }
        if amt < minSend { amt = 0 } // skip tiny transfers
        remaining -= amt
        results = append(results, AllocationResult{Bookie: b, AmountToSend: amt})
    }

    // If rounding left some remainder, distribute it to highest scores
    if remaining > 0 {
        // add to bookies by descending score while respecting max
        for i := 0; remaining > 0 && i < n; i++ {
            // find top remaining index
            topIdx := 0
            for j := 1; j < n; j++ {
                if scores[j] > scores[topIdx] { topIdx = j }
            }
            inc := int64(math.Min(float64(remaining), float64(bookies[topIdx].MaxDeposit - results[topIdx].AmountToSend)))
            if inc <= 0 { break }
            results[topIdx].AmountToSend += inc
            remaining -= inc
            // set score to -inf if can't accept more
            scores[topIdx] = -1
        }
    }

    return results
}
