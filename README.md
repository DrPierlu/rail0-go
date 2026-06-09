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

// List wallets for an account and pick one
wallets, _ := client.Accounts.Wallets(ctx, "account-uuid")
wallet := wallets[0]

// List tokens accepted by that wallet
tokens, _ := client.Wallets.Tokens(ctx, wallet.ID)
usdc := tokens[0] // Token with nested Blockchain

// Create a payment intent
resp, _ := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
    ChainId: usdc.Blockchain.ChainID,
    Mode:    "authorize",
    Amount:  "50000000", // 50 USDC (6 decimals)
    Payer:   "0xBuyer...",
    Payee:   wallet.Address,
    Token:   usdc.Address,
})
```

## Client services

| Service | Description |
|---------|-------------|
| `client.Accounts` | Wallet CRUD scoped to an account |
| `client.Wallets` | Wallet token queries |
| `client.Payments` | Full payment lifecycle |
| `client.Auth` | SIWE authentication |
| `client.Chains` | Blockchain catalog |
| `client.Tokens` | Flat token catalog |

## Accounts

```go
// List wallets for an account (no auth required)
wallets, _ := client.Accounts.Wallets(ctx, accountID)
wallets, _ := client.Accounts.Wallets(ctx, accountID, rail0.ListWalletsParams{
    PerPage: 50,
})

// Get a single wallet
wallet, _ := client.Accounts.GetWallet(ctx, accountID, walletID)

// Create a wallet (requires JWT)
wallet, _ := client.Accounts.CreateWallet(ctx, accountID, rail0.CreateWalletRequest{
    Address: "0xABC...",
    Label:   "Treasury",
})

// Update a wallet (requires JWT)
active := false
wallet, _ = client.Accounts.UpdateWallet(ctx, accountID, walletID, rail0.UpdateWalletRequest{
    Label:  "Archived",
    Active: &active,
})

// Delete a wallet (requires JWT)
err := client.Accounts.DeleteWallet(ctx, accountID, walletID)
```

## Wallets

```go
// List tokens accepted by a wallet
tokens, _ := client.Wallets.Tokens(ctx, walletID)
tokens, _ := client.Wallets.Tokens(ctx, walletID, rail0.ListWalletTokensParams{
    Symbol: "USDC",
})
// tokens[0] is a Token with a nested Blockchain object
// tokens[0].Blockchain.ChainID, .Name, .Slug, .RpcURL
```

## Chains and Tokens

```go
// All supported blockchains
chains, _ := client.Chains.List(ctx)
// chains[0].ChainID, .Name, .Slug, .NativeSymbol, .RpcURL, .ExplorerURL

// Flat token catalog
tokens, _ := client.Tokens.List(ctx)
// tokens[0] is a CatalogToken: .ChainID, .ChainSlug, .Symbol, .Address, .Decimals
```

## Payments

```go
// List (requires JWT)
payments, _ := client.Payments.List(ctx, rail0.ListPaymentsParams{
    Sort: "-created_at",
})

// Get
p, _ := client.Payments.Get(ctx, rail0ID)

// Create
resp, _ := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
    ChainId: 5042002,
    Mode:    "authorize",
    Amount:  "50000000",
    Payer:   "0xBuyer...",
    Payee:   "0xMerchant...",
    Token:   "0xUSDC...",
})

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
