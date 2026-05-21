# rail0-go

Go SDK for the [RAIL0](https://github.com/commercelayer/rail0) stablecoin payment API.

RAIL0 is an immutable smart contract that brings the authorize → capture → refund lifecycle of card networks to stablecoin payments — no intermediaries, no protocol fees, no permission required. This SDK wraps the REST API that sits in front of the contract, giving you fully-typed access to every operation.

## Requirements

- Go ≥ 1.22

## Installation

```bash
go get github.com/rail0/go-sdk
```

## Quick start

```go
package main

import (
    "context"
    "fmt"

    rail0 "github.com/rail0/go-sdk"
)

func main() {
    client := rail0.NewClient(rail0.ClientOptions{
        BaseURL: "https://api.rail0.xyz",
    })
    ctx := context.Background()

    // Step 1 — discover payment methods
    methods, _ := client.Merchants.PaymentMethods(ctx, 1)
    usdc := methods[0] // pick USDC on the target chain

    // Step 2 — create payment intent
    resp, _ := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
        Payment: rail0.PaymentConfig{
            Payer: "0xBuyer...",
            Payee: usdc.WalletAddress,
            Token: usdc.TokenAddress,
        },
        Amount:  "50000000", // 50 USDC (6 decimals)
        ChainID: int64(usdc.ChainID),
        Mode:    "authorize",
    })

    // Step 3 — payer signs the EIP-3009 payload off-chain
    key, _ := rail0.HexToPrivateKey("0x...")
    sig, _ := rail0.SignAuthorize(rail0.SignPaymentParams{
        PrivateKey:      key,
        Payment:         resp.Payment,
        Amount:          resp.Amount,
        Nonce:           resp.SigningPayload.Message.Nonce,
        ContractAddress: resp.Rail0Contract,
        TokenDomain: rail0.TokenDomain{
            Name:              "USD Coin",
            Version:           "2",
            ChainID:           int64(usdc.ChainID),
            VerifyingContract: usdc.TokenAddress,
        },
    })

    // Step 4 — submit payer signature
    client.Payments.Sign(ctx, resp.PaymentID, rail0.PayerSignatureRequest{
        V: sig.V, R: sig.R, S: sig.S,
    })

    // Step 5 — payee prepares the unsigned authorize tx
    tx, _ := client.Payments.Authorize(ctx, resp.PaymentID)
    // sign tx.UnsignedTransaction with payee's EIP-1559 key

    // Step 6 — broadcast signed authorize tx
    client.Payments.SubmitAuthorize(ctx, resp.PaymentID, rail0.SubmitTransactionRequest{
        SignedTransaction: signedBytes,
    })

    fmt.Println("authorized:", resp.PaymentID)
}
```

## Payment lifecycle

```text
                            authorizationExpiry       refundExpiry
                                   │                       │
  ─────────────────────────────────┼───────────────────────┼──────▶ time
   create → sign → authorize       │   capture / void       │   approve+refund
                                    │   release              │
```

| Operation | Caller | What it does |
|-----------|--------|--------------|
| `Authorize` + `SubmitAuthorize` | payee | Prepare + broadcast the authorize tx; funds move to escrow |
| `Charge` | payee | Server-side one-shot: authorize + capture with no escrow window |
| `PrepareCapture` + `SubmitCapture` | payee | Moves escrowed funds to the merchant |
| `PrepareVoid` + `SubmitVoid` | payee | Cancels the hold, returns funds to the payer |
| `PrepareRelease` + `SubmitRelease` | anyone | Reclaims escrow after `AuthorizationExpiry` |
| `PrepareApprove` + `SubmitApprove` | payee | ERC-20 `approve()` required before a refund |
| `PrepareRefund` + `SubmitRefund` | payee | Returns captured funds to the payer |

## API reference

### `rail0.NewClient(opts)`

```go
client := rail0.NewClient(rail0.ClientOptions{
    BaseURL:    "https://api.rail0.xyz",
    Headers:    map[string]string{"Authorization": "Bearer ..."},
    Timeout:    30 * time.Second,       // default 30s
    MaxRetries: 3,                      // default 0 (no retry)
    RetryDelay: 200 * time.Millisecond, // base delay, doubles each attempt
    Logger:     rail0.DebugLogger,      // optional
})
```

---

### Logging

Pass any `func(rail0.LogEntry)` as `Logger` to receive structured log entries.

```go
// Built-in logger — writes one line per request to stdout
client := rail0.NewClient(rail0.ClientOptions{
    BaseURL: "https://api.rail0.xyz",
    Logger:  rail0.DebugLogger,
})
```

Output:
```text
[rail0] POST 200 https://.../payments 87ms
[rail0] ERROR PUT https://.../payments/0x.../sign 30001ms
```

To integrate with `slog`, `zap`, or `zerolog`:

```go
logger := slog.Default()
client := rail0.NewClient(rail0.ClientOptions{
    Logger: func(e rail0.LogEntry) {
        if e.Err != nil {
            logger.Error("rail0 request failed", "method", e.Method, "url", e.URL, "err", e.Err)
        } else {
            logger.Debug("rail0 request", "method", e.Method, "status", e.Status, "ms", e.DurationMs)
        }
    },
})
```

---

### `client.Merchants`

#### `.PaymentMethods(ctx, merchantID)` → `([]PaymentMethod, error)`

Returns the active payment methods (chain + token + wallet) for a merchant.

```go
methods, err := client.Merchants.PaymentMethods(ctx, 1)
// methods[0].ChainID, .TokenAddress, .WalletAddress, .TokenSymbol, .ChainSlug
```

---

### `client.Payments`

All methods take `ctx context.Context` and return `(*T, error)`.

#### `.Get(ctx, paymentID)` → `*PaymentResponse`

Fetches the current payment state (DB status + live on-chain escrow balances).

```go
res, _ := client.Payments.Get(ctx, paymentID)
// res.Status, res.OnChain.CapturableAmount, res.OnChain.RefundableAmount
```

#### `.CreatePayment(ctx, params)` → `*CreatePaymentResponse`

Creates a payment intent. Returns `SigningPayload` for the payer to sign, plus `Rail0Contract`.

#### `.Sign(ctx, paymentID, params)` → `*PayerSignatureResponse`

Submits the payer's EIP-712 signature (v, r, s).

#### `.Authorize(ctx, paymentID)` → `*PrepareTransactionResponse`

Prepares the unsigned `authorize()` transaction. Called by the payee. Sign `UnsignedTransaction` with the payee's key and pass to `SubmitAuthorize`.

#### `.SubmitAuthorize(ctx, paymentID, params)` → `*AuthorizePaymentResponse`

Broadcasts the signed authorize transaction. Funds are moved to escrow.

```go
tx, _ := client.Payments.Authorize(ctx, paymentID)
res, _ := client.Payments.SubmitAuthorize(ctx, paymentID, rail0.SubmitTransactionRequest{
    SignedTransaction: signedBytes,
})
// res.TransactionHash, res.CapturableAmount
```

#### `.Charge(ctx, paymentID)` → `*ChargePaymentResponse`

Server-side one-shot: authorize + capture in a single transaction. No `Submit` step. Called by the payee.

#### `.PrepareCapture(ctx, paymentID, params)` / `.SubmitCapture(ctx, paymentID, params)`

Build and broadcast the capture transaction. Partial captures are supported.

```go
tx, _ := client.Payments.PrepareCapture(ctx, paymentID, rail0.CapturePaymentRequest{Amount: "50000000"})
res, _ := client.Payments.SubmitCapture(ctx, paymentID, rail0.SubmitTransactionRequest{SignedTransaction: signed})
// res.CapturedAmount, res.CapturableAmount, res.RefundableAmount
```

#### `.PrepareVoid(ctx, paymentID)` / `.SubmitVoid(ctx, paymentID, params)`

Void the authorization — releases all escrowed funds to the payer.

#### `.PrepareRelease(ctx, paymentID, params)` / `.SubmitRelease(ctx, paymentID, params)`

Release escrowed funds after `AuthorizationExpiry`. Set `CallerAddress` in `ReleaseRequest` to build the tx for the buyer.

```go
tx, _ := client.Payments.PrepareRelease(ctx, paymentID, rail0.ReleaseRequest{CallerAddress: buyerAddr})
client.Payments.SubmitRelease(ctx, paymentID, rail0.SubmitTransactionRequest{SignedTransaction: buyerSigned})
```

#### `.PrepareApprove(ctx, paymentID, params)` / `.SubmitApprove(ctx, paymentID, params)`

ERC-20 `approve()` before a refund. Include `Amount` in `SubmitApproveRequest` so the API records it.

```go
tx, _ := client.Payments.PrepareApprove(ctx, paymentID, rail0.ApproveRequest{Amount: "50000000"})
client.Payments.SubmitApprove(ctx, paymentID, rail0.SubmitApproveRequest{
    SignedTransaction: signed, Amount: "50000000",
})
```

#### `.PrepareRefund(ctx, paymentID, params)` / `.SubmitRefund(ctx, paymentID, params)`

Build and broadcast the refund transaction. Partial refunds are supported.

---

## Off-chain signing

RAIL0 uses EIP-3009 `transferWithAuthorization` — the payer signs off-chain and the API submits on their behalf (gasless for the payer).

```go
key, _ := rail0.HexToPrivateKey("0xYourPrivateKey...")
sig, err := rail0.SignAuthorize(rail0.SignPaymentParams{
    PrivateKey:      key,
    Payment:         resp.Payment,
    Amount:          resp.Amount,
    Nonce:           resp.SigningPayload.Message.Nonce,
    ContractAddress: resp.Rail0Contract,
    TokenDomain: rail0.TokenDomain{
        Name:              "USD Coin",
        Version:           "2",
        ChainID:           84532,
        VerifyingContract: token,
    },
})
// sig.V, sig.R, sig.S — pass to Sign()
```

Use `SignCharge` instead of `SignAuthorize` when `mode: "charge"`.

---

### Stablecoin registry

```go
// All EIP-3009 tokens on Base (compatible with RAIL0)
tokens := rail0.EIP3009Tokens("base")
// tokens[0]: { Symbol: "USDC", Address: "0x833...", Decimals: 6 }

// Chain metadata
info := rail0.ChainInfo("base")
fmt.Println(info.ChainID) // 8453
```

---

## Error handling

Every non-2xx response is returned as `*APIError`:

```go
import "errors"

_, err := client.Payments.SubmitCapture(ctx, paymentID, params)
if err != nil {
    var apiErr *rail0.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.Status, apiErr.Code, apiErr.Message)
    }
    return err
}
```

Common error codes:

| Code | Cause |
|------|-------|
| `PaymentAlreadyExists` | `Authorize`/`Charge` called twice with the same `paymentId` |
| `PaymentNotFound` | `paymentId` does not exist |
| `AuthorizationExpired` | `AuthorizationExpiry` is in the past (capture) |
| `AuthorizationNotExpired` | `AuthorizationExpiry` has not passed yet (release) |
| `RefundExpired` | `RefundExpiry` is in the past |
| `InvalidAmount` | `amount` is 0 |
| `NotPayee` | caller is not `payment.Payee` |

---

## Development

```bash
go test ./...

# Regenerate types_gen.go after an API change:
# 1. Update the schema in rail0-api (sibling repo),
#    or set RAIL0_SCHEMA_PATH to point to a local file.
# 2. Regenerate:
go run gen/generate.go
go build ./...
```

## Project structure

```text
*.go              package rail0 — SDK source
  client.go       Client struct and NewClient
  merchants.go    MerchantsService
  payments.go     PaymentsService
  types.go        public types (hand-documented)
  types_gen.go    generated types (never hand-edited)
  signing.go      EIP-712/EIP-3009 off-chain signing
  stablecoins.go  stablecoin address registry
  http.go         internal HTTP client (retry, logging)
  error.go        *APIError

gen/
  generate.go     generates types_gen.go from the schema

go.mod / go.sum
README.md
```

---

## License

[MIT](LICENSE)
