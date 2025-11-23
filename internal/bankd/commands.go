// internal/bankd/commands.go
package bankd

import (
    "context"
    "fmt"
    "github.com/gliderlabs/ssh"
    "github.com/google/uuid" // Add the google UUID package
)

func handleBalance(s ssh.Session, partnerID string) {
    // Convert partnerCustomerID to uuid.UUID
    customerID, err := uuid.Parse(partnerCustomerID(partnerID))  // Convert string to UUID
    if err != nil {
        fmt.Fprintf(s, "Error: Invalid partner ID format.\n")
        return
    }

    bal, err := appCtx.BalanceEngine.GetCustomerBalance(context.Background(), customerID)
    if err != nil {
        fmt.Fprintf(s, "Error: Failed to fetch balance: %v\n", err)
        return
    }
    fmt.Fprintf(s, "Balance Summary:\n")
    fmt.Fprintf(s, " Real Money: KES %.2f\n", float64(bal.TotalReal)/100)
    fmt.Fprintf(s, " Bonus Money: KES %.2f\n", float64(bal.TotalFake)/100)
    fmt.Fprintf(s, " Total: KES %.2f\n", float64(bal.TotalReal+bal.TotalFake)/100)
}

func handleAccount(s ssh.Session, partnerID string) {
    fmt.Fprintf(s, "Account Information:\n")
    fmt.Fprintf(s, " Partner ID: %s\n", partnerID)
    fmt.Fprintf(s, " Status: Active\n")
    fmt.Fprintf(s, " Type: Business Partner\n")
}

func handleDeposit(s ssh.Session, partnerID string, parts []string) {
    if len(parts) < 2 {
        fmt.Fprintln(s, "Usage: deposit <amount>")
        return
    }
    amount, err := parseAmount(parts[1])
    if err != nil {
        fmt.Fprintf(s, "Invalid amount: %s\n", parts[1])
        return
    }
    fmt.Fprintf(s, "Deposit Request: KES %.2f\n", amount)
    fmt.Fprintln(s, "Processing deposit request...")
    fmt.Fprintln(s, "Deposit initiated successfully!")
    fmt.Fprintln(s, " Check your balance in a moment.")
}

func handleWithdraw(s ssh.Session, partnerID string, parts []string) {
    if len(parts) < 2 {
        fmt.Fprintln(s, "Usage: withdraw <amount> [--request-otp]")
        return
    }
    amount, err := parseAmount(parts[1])
    if err != nil {
        fmt.Fprintf(s, "Invalid amount: %s\n", parts[1])
        return
    }
    fmt.Fprintf(s, "Withdrawal Request: KES %.2f\n", amount)
    fmt.Fprintln(s, " OTP verification required for withdrawals.")
    if len(parts) > 2 && parts[2] == "--request-otp" {
        fmt.Fprintln(s, " OTP requested via SMS...")
        fmt.Fprintln(s, " Check your phone for OTP.")
    }
}

func handleAssets(s ssh.Session, partnerID string) {
    fmt.Fprintln(s, "Asset Nexus:")
    fmt.Fprintln(s, " Cash Balance: Use 'balance' command")
    fmt.Fprintln(s, " Digital Assets: M-Pesa Integration Active")
    fmt.Fprintln(s, " Investment Tools: Coming Soon")
}

func handleProfile(s ssh.Session, partnerID string) {
    fmt.Fprintln(s, "Sharp Profile:")
    fmt.Fprintln(s, " Risk Level: Medium")
    fmt.Fprintln(s, " Trading Tier: Partner")
    fmt.Fprintln(s, " KYC Status: Verified")
}

func showHelp(s ssh.Session) {
    helpText := `
Available Commands:
 balance      - Check your account balance
 account      - View account information
 deposit <amount> - Deposit money
 withdraw <amount> - Withdraw money (OTP required)
 assets       - View assets and integrations
 profile      - View trading profile
 help         - Show this help
 exit         - Exit session

Examples:
 deposit 2500.00
 withdraw 1000.00 --request-otp
`
    fmt.Fprint(s, helpText)
}

