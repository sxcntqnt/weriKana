# money-transfer
Attempt to translate the monolith into Go before ripping it into pieces
.
├── main.go
├── db/
│   └── initdb.go
├── models/
│   ├── bank.go
│   ├── bookie.go
│   ├── customer.go
│   ├── sender.go
│   ├── recipient.go
│   ├── bookie_account.go
│   └── transaction.go
├── api/handlers/
│   ├── smart-depo.go
│   ├── smart-withdraw.go
│   ├── otp.go
│   └── fake-topup.go
├── services/
│   ├── natsclient/
│   ├── mpesa/
│   ├── keystore/
│   └── otp/
└── go.mod
