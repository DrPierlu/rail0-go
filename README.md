# rail0-go

Go SDK for the [RAIL0](https://github.com/rail0/rail0) stablecoin payment API.

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
    "math/big"
    "time"

    rail0 "github.com/rail0/go-sdk"
)

func main() {
    client := rail0.NewClient(rail0.ClientOptions{
        BaseURL: "https://api.rail0.xyz",
    })

    ctx := context.Background()
    now := time.Now().Unix()

    payment := rail0.Payment{
        Payer:               "0xBuyer...",
        Payee:               "0xMerchant...",
        Token:               "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
        Amount:              "100000000",                                    // 100 USDC (6 decimals)
        AuthorizationExpiry: now + 3600*24,                                 // 24 h to capture
        RefundExpiry:        now + 3600*24*7,                               // 7-day refund window
        FeeBps:              0,
        FeeReceiver:         "0x0000000000000000000000000000000000000000",
    }

    paymentID := "0xabc..." // your unique identifier for this payment

    // Step 1 — get the nonce for the payer's EIP-3009 signature
    nonceResp, _ := client.Payments.AuthorizeNonce(ctx, paymentID, payment.Payer)

    // Step 2 — sign off-chain (payer's private key never leaves the client)
    key, _ := rail0.HexToPrivateKey("0x...") // payer's private key
    sig, _ := rail0.SignAuthorize(rail0.SignPaymentParams{
        PrivateKey:      key,
        Payment:         payment,
        Amount:          big.NewInt(50_000_000), // 50 USDC
        Nonce:           nonceResp.Nonce,
        ContractAddress: "0xRAIL0ContractAddress...",
        TokenDomain: rail0.TokenDomain{
            Name:              "USD Coin",
            Version:           "2",
            ChainID:           8453,
            VerifyingContract: payment.Token,
        },
    })

    // Step 3 — payer locks funds in escrow
    _, _ = client.Payments.Authorize(ctx, paymentID, rail0.AuthorizeParams{
        Payment: payment,
        Amount:  "50000000",
        V:       sig.V,
        R:       sig.R,
        S:       sig.S,
    })

    // Step 4 — merchant releases them
    tx, _ := client.Payments.Capture(ctx, paymentID, rail0.CaptureParams{
        Payment: payment,
        Amount:  "50000000",
    })

    fmt.Println(tx.TransactionHash, tx.Status)
}
```

## Payment lifecycle

```text
                        authorizationExpiry         refundExpiry
                               │                         │
  ─────────────────────────────┼─────────────────────────┼──────▶ time
   authorize / charge           │   capture / void         │   refund
                                │   release (permissionless)
```

| Operation | Caller | What it does |
|-----------|--------|--------------|
| `Authorize` | payer | Locks `amount` in escrow via EIP-3009 signature |
| `Charge` | payer | Authorize + capture in one transaction |
| `Capture` | payee | Moves escrowed funds to the merchant |
| `Void` | payee | Cancels the hold, returns funds to the payer |
| `Release` | anyone | Reclaims escrow after `authorizationExpiry` with no capture |
| `Refund` | payee | Returns previously captured funds to the payer |

## API reference

### `rail0.NewClient(opts)`

```go
client := rail0.NewClient(rail0.ClientOptions{
    BaseURL:    "https://api.rail0.xyz",
    Headers:    map[string]string{"Authorization": "Bearer ..."},
    Timeout:    30 * time.Second,     // default 30s
    MaxRetries: 3,                    // default 0 (no retry)
    RetryDelay: 200 * time.Millisecond, // base delay, doubles each attempt
    Logger:     rail0.DebugLogger,    // optional
    Transport:  myTransport,          // optional — custom http.RoundTripper
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
[rail0] POST 202 http://... /payments/0x.../authorize 87ms
[rail0] ERROR GET http://... /payments/0x... 30001ms
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

`LogEntry` fields:

| Field | Type | Present |
|-------|------|---------|
| `Method` | `string` | always |
| `URL` | `string` | always |
| `DurationMs` | `int64` | always |
| `RequestBody` | `any` | POST requests |
| `Status` | `int` | when a response was received (0 on network error) |
| `ResponseBody` | `any` | when a response was received |
| `Err` | `error` | on HTTP errors and network failures |
| `Attempt` | `int` | when `MaxRetries > 0` |
| `WillRetry` | `bool` | when `MaxRetries > 0` and a retry is scheduled |

---

### `client.Payments`

All methods take `ctx context.Context` as first argument and return `(*T, error)`.

#### `.Get(ctx, paymentID)`

Returns the on-chain state and config hash for a payment.

```go
res, err := client.Payments.Get(ctx, paymentID)
// res.State: { Exists, CapturableAmount, RefundableAmount }
// res.ConfigHash: EIP-712 digest committed on creation
```

#### `.Authorize(ctx, paymentID, params)`

Locks `Amount` from the payer into escrow. Build the EIP-3009 signature with `SignAuthorize`.

#### `.Charge(ctx, paymentID, params)`

Authorize and capture in one transaction. Build the EIP-3009 signature with `SignCharge`.

#### `.Capture(ctx, paymentID, params)` / `.Void(ctx, paymentID, params)`

Capture escrowed funds or void (return them to payer). Caller must be the payee.

#### `.Release(ctx, paymentID, params)`

Return escrowed funds to the payer after `AuthorizationExpiry`. Permissionless.

#### `.Refund(ctx, paymentID, params)`

Return a previously captured amount to the payer. Must be called before `RefundExpiry`.

#### `.AuthorizeNonce(ctx, paymentID, payer)` / `.ChargeNonce(ctx, paymentID, payer)`

Returns the EIP-3009 nonce to include in the payer's signature.

#### `.Hash(ctx, payment)`

Computes the EIP-712 digest of a `Payment` configuration.

---

### `client.Tokens`

#### `.IsAccepted(ctx, address)`

Returns whether the given ERC-20 token is in this deployment's allowlist.

---

### `client.Utils`

#### `.DomainSeparator(ctx)`

Returns the EIP-712 domain separator for the RAIL0 contract.

#### `.Version(ctx)`

Returns the contract version number.

---

### Off-chain signing

RAIL0 uses EIP-3009 `transferWithAuthorization` — the payer signs a payload off-chain and the API submits the transaction on their behalf (gasless for the payer).

```go
// 1. Get the nonce for this (paymentId, payer) pair
nonce, _ := client.Payments.AuthorizeNonce(ctx, paymentID, payment.Payer)

// 2. Sign
key, _ := rail0.HexToPrivateKey("0xYourPrivateKey...")
sig, err := rail0.SignAuthorize(rail0.SignPaymentParams{
    PrivateKey:      key,
    Payment:         payment,
    Amount:          big.NewInt(50_000_000),
    Nonce:           nonce.Nonce,
    ContractAddress: "0xRAIL0...",
    TokenDomain: rail0.TokenDomain{
        Name:              "USD Coin",
        Version:           "2",
        ChainID:           8453,
        VerifyingContract: payment.Token,
    },
})

// 3. Submit
client.Payments.Authorize(ctx, paymentID, rail0.AuthorizeParams{
    Payment: payment,
    Amount:  "50000000",
    V: sig.V, R: sig.R, S: sig.S,
})
```

`SignCharge` works the same way — use `.ChargeNonce` to obtain the nonce.

For raw control use `SignTransferWithAuthorization`:

```go
sig, err := rail0.SignTransferWithAuthorization(key, domain, rail0.SignTransferParams{
    From:        payment.Payer,
    To:          contractAddress,
    Value:       big.NewInt(50_000_000),
    ValidAfter:  new(big.Int), // 0 = immediate
    ValidBefore: big.NewInt(payment.AuthorizationExpiry),
    Nonce:       nonce.Nonce,
})
```

---

### Stablecoin registry

```go
// All EIP-3009 tokens on Base (compatible with RAIL0)
tokens := rail0.EIP3009Tokens("base")
// tokens[0]: { Symbol: "USDC", Address: "0x833...", Decimals: 6 }

// All chains
for chain, info := range rail0.Stablecoins {
    fmt.Printf("%s (chainId %d): %d tokens\n", chain, info.ChainID, len(info.Tokens))
}
```

---

### Error handling

Every non-2xx response is returned as `*APIError`:

```go
import "errors"

tx, err := client.Payments.Capture(ctx, paymentID, params)
if err != nil {
    var apiErr *rail0.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.Status)  // 422
        fmt.Println(apiErr.Code)    // "AuthorizationExpired"
        fmt.Println(apiErr.Message) // human-readable
    }
    return err
}
```

Common error codes:

| Code | Cause |
|------|-------|
| `PaymentAlreadyExists` | `Authorize`/`Charge` called twice with the same `paymentId` |
| `PaymentNotFound` | `paymentId` does not exist |
| `PaymentMismatch` | `payment` config does not match the stored hash |
| `AuthorizationExpired` | `AuthorizationExpiry` is in the past (Capture) |
| `AuthorizationNotExpired` | `AuthorizationExpiry` has not passed yet (Release) |
| `RefundExpired` | `RefundExpiry` is in the past |
| `InvalidAmount` | `amount` is 0 |
| `TokenNotAccepted` | token is not in this deployment's allowlist |
| `NotPayee` | caller is not `payment.Payee` |

---

## Development

### Run tests

```bash
go test ./...
```

### Regenerate types after an API change

```bash
# 1. Update the schema in rail0-api (sibling repo),
#    or set RAIL0_SCHEMA_PATH to point to a local file.

# 2. Regenerate types_gen.go
go run gen/generate.go

# 3. Check for breakage
go build ./...
go test ./...
```

---

## Project structure

```text
gen/              Generation pipeline (schema from rail0-api)
  generate.go     generates types_gen.go from the schema

test/             test suite
  signing_test.go signing utility tests (EIP-712 cross-check)
  client_test.go  HTTP client tests (retry, logging, error handling)
  integration_test.go  endpoint shape tests (httptest mock server)

*.go              package rail0 — SDK source
  client.go       Client struct and NewClient
  types.go        public types (hand-documented)
  error.go        *APIError
  http.go         internal HTTP client (retry, logging)
  payments.go     PaymentsService
  tokens.go       TokensService
  utils.go        UtilsService
  signing.go      EIP-712/EIP-3009 off-chain signing
  stablecoins.go  stablecoin address registry

go.mod / go.sum   module definition
README.md
```

---

## License

[MIT](LICENSE)
