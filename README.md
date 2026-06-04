# rail0-go

Go SDK for the [RAIL0](https://github.com/commercelayer/rail0) stablecoin payment API.

RAIL0 is an immutable EVM smart contract that brings the authorize → capture → refund lifecycle of card networks to stablecoin payments — no intermediaries, no protocol fees, no permission required. This SDK wraps the REST API that sits in front of the contract.

## Requirements

Go ≥ 1.22

## Installation

```bash
go get github.com/rail0/go-sdk
```

## Quick start

```go
client := rail0.NewClient(rail0.ClientOptions{
    BaseURL: "https://api.rail0.xyz",
    Headers: map[string]string{"Authorization": "Bearer <jwt>"},
})
ctx := context.Background()

// Discover the account's wallets and pick a payment method
wallets, _ := client.Accounts.Wallets(ctx, "account-uuid")
w := wallets.Data[0]
token := w.Tokens[0]

// Create a payment intent
resp, _ := client.Payments.Create(ctx, rail0.CreatePaymentRequest{
    Payment: rail0.PaymentInput{
        Payer:  "0xBuyer...",
        Payee:  w.Address,
        Token:  token.TokenAddress,
        Amount: "50000000", // 50 USDC (6 decimals)
    },
    ChainId: token.ChainID,
    Mode:    "authorize",
})
```

## Client services

| Service | Description |
|---------|-------------|
| `client.Accounts` | Account profile, wallet listing, merchant payments |
| `client.Wallets` | Wallet and wallet-token management (requires JWT) |
| `client.Payments` | Full payment lifecycle |
| `client.Auth` | SIWE authentication |
| `client.Chains` | Blockchain catalog |
| `client.Tokens` | Token catalog |

## Accounts

```go
// Public — list wallets with nested tokens
wallets, _ := client.Accounts.Wallets(ctx, accountID, rail0.ListWalletsParams{
    Sort:    "-created_at",
    PerPage: 50,
})

// Protected — account profile
account, _ := client.Accounts.Get(ctx, accountID)

// Protected — update profile
account, _ := client.Accounts.Update(ctx, accountID, rail0.UpdateAccountRequest{
    Email: "new@example.com",
})

// Protected — payments where this account is payee
payments, _ := client.Accounts.Payments(ctx, accountID, rail0.ListAccountPaymentsParams{
    Status: "authorized",
    Sort:   "-created_at,amount",
})
```

## Wallets (protected)

```go
// List wallets with tokens
wallet, _ := client.Wallets.Get(ctx, walletID)

// Create
w, _ := client.Wallets.Create(ctx, rail0.CreateWalletRequest{
    Address: "0xABC...",
    Label:   "Treasury",
})

// Update
client.Wallets.Update(ctx, walletID, rail0.UpdateWalletRequest{Label: "Main"})

// Add a token
client.Wallets.AddToken(ctx, walletID, rail0.AddTokenRequest{TokenID: tokenID})

// Update a token
active := false
client.Wallets.UpdateToken(ctx, walletID, tokenID, rail0.UpdateTokenRequest{Active: &active})

// Remove a token (hard delete)
client.Wallets.RemoveToken(ctx, walletID, tokenID)
```

## Payments

```go
// List (requires JWT)
payments, _ := client.Payments.List(ctx, rail0.ListPaymentsParams{
    Status: "authorized",
    Sort:   "-created_at",
})

// Get
p, _ := client.Payments.Get(ctx, rail0ID)

// Create
resp, _ := client.Payments.Create(ctx, rail0.CreatePaymentRequest{...})

// Submit payer signature
client.Payments.Sign(ctx, rail0ID, rail0.PayerSignatureRequest{Signature: "0x..."})

// Authorize → Capture
tx, _ := client.Payments.AuthorizePrepare(ctx, rail0ID)
// sign tx.UnsignedTransaction → signedBytes
client.Payments.Authorize(ctx, rail0ID, rail0.SubmitTransactionRequest{SignedTransaction: signedBytes})

tx, _ = client.Payments.CapturePrepare(ctx, rail0ID, rail0.CapturePaymentRequest{Amount: "50000000"})
// sign → signedBytes
client.Payments.Capture(ctx, rail0ID, rail0.SubmitTransactionRequest{SignedTransaction: signedBytes})

// Transactions for a payment
txs, _ := client.Payments.Transactions(ctx, rail0ID, rail0.ListTransactionsParams{
    Sort: "-submitted_at",
})
```

## Sorting

List endpoints accept a `Sort` field using JSON:API convention:

- `"created_at"` — ascending
- `"-created_at"` — descending
- `"-created_at,amount"` — multiple fields, left-to-right priority

| Method | Allowed sort fields |
|--------|-------------------|
| `Accounts.Wallets` | `created_at`, `label`, `address` |
| `Accounts.Payments` | `created_at`, `amount`, `status`, `authorization_expiry` |
| `Payments.List` | `created_at`, `amount`, `status`, `authorization_expiry` |
| `Payments.Transactions` | `submitted_at`, `confirmed_at`, `operation` |

## Authentication (SIWE)

```go
// 1. Get a nonce
nonce, _ := client.Auth.GetNonce(ctx)

// 2. Build + sign the EIP-4361 message with your wallet, then:
resp, _ := client.Auth.Verify(ctx, rail0.VerifyRequest{
    Message:   siweMessage,
    Signature: "0x...",
})
// resp.Token — include as Authorization: Bearer <token>
```

## Configuration

```go
client := rail0.NewClient(rail0.ClientOptions{
    BaseURL:    "https://api.rail0.xyz",
    Headers:    map[string]string{"Authorization": "Bearer " + token},
    Timeout:    30 * time.Second,
    MaxRetries: 3,
    RetryDelay: 200 * time.Millisecond,
    Logger:     rail0.DebugLogger,
})
```

## Error handling

```go
_, err := client.Payments.Authorize(ctx, rail0ID, params)
if err != nil {
    var apiErr *rail0.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.Status, apiErr.Code, apiErr.Message)
    }
}
```

## Development

```bash
go test ./...

# Regenerate after an API schema change:
RAIL0_SCHEMA_PATH=../rail0-api/docs/openapi.json go run gen/generate.go
go build ./...
```

## License

[MIT](LICENSE)
