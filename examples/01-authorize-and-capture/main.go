// Standard two-step payment flow: authorize → capture
//
// The buyer locks funds in escrow using an EIP-3009 signature (authorize).
// The merchant releases them once the order is fulfilled (capture).
// If something goes wrong before capture the merchant can void,
// or anyone can call release after authorizationExpiry.
//
// On-chain flow:
//
//	buyer signs EIP-3009 → Authorize()  funds move buyer → escrow
//	merchant             → Capture()    funds move escrow → merchant (minus fee)
//	merchant             → Void()       alternative: funds move escrow → buyer
//	anyone               → Release()    fallback after authorizationExpiry
//
// Run:
//
//	go run examples/01-authorize-and-capture/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	rail0 "github.com/rail0/go-sdk"
)

func main() {
	ctx := context.Background()
	client := rail0.NewClient(rail0.ClientOptions{
		BaseURL: "https://api.rail0.xyz",
	})

	// ----------------------------------------------------------------
	// Shared payment configuration
	// A unique ID for this payment — in practice derive it from your
	// order ID, e.g. keccak256(abi.encode("order", orderId)).
	// ----------------------------------------------------------------

	const paymentID = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	now := time.Now().Unix()

	payment := rail0.Payment{
		Payer:               "0xBuyerAddress000000000000000000000000000000",
		Payee:               "0xMerchantAddress0000000000000000000000000000",
		Token:               "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
		MaxAmount:           "100000000",                                    // 100 USDC (6 decimals)
		AuthorizationExpiry: now + 60*60*24,                                // merchant has 24 h to capture
		RefundExpiry:        now + 60*60*24*7,                              // refund window: 7 days
		FeeBps:              50,                                            // 0.5% protocol fee
		FeeReceiver:         "0xFeeReceiverAddress000000000000000000000000",
	}

	// ----------------------------------------------------------------
	// Step 1 — Buyer fetches the authorize nonce, signs EIP-3009, calls Authorize
	// ----------------------------------------------------------------

	// Fetch the nonce the EIP-3009 signature must use.
	nonceResp, err := client.Payments.AuthorizeNonce(ctx, paymentID, payment.Payer)
	if err != nil {
		log.Fatalf("AuthorizeNonce: %v", err)
	}

	// The buyer builds and signs transferWithAuthorization off-chain.
	// In production this happens in the buyer's wallet or using SignAuthorize:
	//
	//   key, _ := rail0.HexToPrivateKey("0xYourPrivateKey")
	//   sig, _ := rail0.SignAuthorize(rail0.SignPaymentParams{
	//       PrivateKey:      key,
	//       Payment:         payment,
	//       Amount:          big.NewInt(50_000_000),
	//       Nonce:           nonceResp.Nonce,
	//       ContractAddress: "0xRAIL0ContractAddress",
	//       TokenDomain: rail0.TokenDomain{
	//           Name: "USD Coin", Version: "2", ChainID: 8453,
	//           VerifyingContract: payment.Token,
	//       },
	//   })

	_ = big.NewInt(50_000_000) // amount we're authorizing: 50 USDC

	authTx, err := client.Payments.Authorize(ctx, paymentID, rail0.AuthorizeParams{
		Payment: payment,
		Amount:  "50000000", // 50 USDC
		V:       27,         // from signature
		R:       "0x1111111111111111111111111111111111111111111111111111111111111111",
		S:       "0x2222222222222222222222222222222222222222222222222222222222222222",
	})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			// Common: TokenNotAccepted, InvalidAmount, PaymentAlreadyExists
			log.Fatalf("Authorize failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Authorize: %v", err)
	}

	fmt.Printf("Authorized: %s — status: %s\n", authTx.TransactionHash, authTx.Status)
	fmt.Printf("Nonce used: %s\n", nonceResp.Nonce)

	// ----------------------------------------------------------------
	// Step 2a — Merchant captures 50 USDC (happy path)
	// ----------------------------------------------------------------

	captureTx, err := client.Payments.Capture(ctx, paymentID, rail0.CaptureParams{
		Payment: payment,
		Amount:  "50000000",
	})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			// Common: AuthorizationExpired, InvalidCaptureAmount, PaymentMismatch
			log.Fatalf("Capture failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Capture: %v", err)
	}

	fmt.Printf("Captured: %s\n", captureTx.TransactionHash)

	// ----------------------------------------------------------------
	// Step 2b — Merchant voids (alternative: order cancelled)
	// ----------------------------------------------------------------

	// voidTx, err := client.Payments.Void(ctx, paymentID, rail0.VoidParams{Payment: payment})

	// ----------------------------------------------------------------
	// Step 2c — Release (fallback: merchant never captured)
	// Only callable after authorizationExpiry. Anyone can call this.
	// ----------------------------------------------------------------

	// releaseTx, err := client.Payments.Release(ctx, paymentID, rail0.ReleaseParams{Payment: payment})

	// ----------------------------------------------------------------
	// Inspect on-chain state at any point
	// ----------------------------------------------------------------

	state, err := client.Payments.Get(ctx, paymentID)
	if err != nil {
		log.Fatalf("Get: %v", err)
	}
	fmt.Printf("Payment state: exists=%v capturable=%s refundable=%s\n",
		state.State.Exists, state.State.CapturableAmount, state.State.RefundableAmount)
	// exists=true capturable=0 refundable=50000000
}
