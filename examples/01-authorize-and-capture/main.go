// Standard two-step payment flow: authorize → capture
//
// The payer creates a payment intent, signs the EIP-712 payload, and
// submits the signature. The payee then calls Authorize (funds move to
// escrow), prepares a capture transaction, signs it offline, and submits it.
//
// On-chain flow:
//
//	payer signs EIP-712    → Authorize()   funds move payer → escrow
//	payee signs capture tx → Capture()     funds move escrow → payee (minus fee)
//	payee signs void tx    → Void()        alternative: funds move escrow → payer
//	anyone                 → Release()     fallback after authorizationExpiry
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
	"time"

	rail0 "github.com/rail0/go-sdk"
)

func main() {
	ctx := context.Background()
	client := rail0.NewClient(rail0.ClientOptions{
		BaseURL: "https://api.rail0.xyz",
	})

	now := time.Now().Unix()

	payment := rail0.PaymentConfig{
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
	// Step 1 — Payer creates a payment intent and signs the EIP-712 payload
	// ----------------------------------------------------------------

	createResp, err := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
		Payment: payment,
		Amount:  "50000000", // 50 USDC
		ChainID: 8453,       // Base
		Mode:    "authorize",
	})
	if err != nil {
		log.Fatalf("CreatePayment: %v", err)
	}

	fmt.Printf("Payment ID: %s\n", createResp.PaymentID)
	fmt.Printf("Config hash: %s\n", createResp.ConfigHash)

	// The payer signs the signingPayload using eth_signTypedData_v4 (wallet)
	// or with SignAuthorize (direct key access):
	//
	//   key, _ := rail0.HexToPrivateKey("0xYourPrivateKey")
	//   sig := rail0.SignAuthorize(rail0.SignPaymentParams{
	//       PrivateKey:      key,
	//       Payment:         payment,
	//       Amount:          big.NewInt(50_000_000),
	//       Nonce:           createResp.SigningPayload.Message.Nonce,
	//       ContractAddress: createResp.Rail0Contract,
	//       TokenDomain: rail0.TokenDomain{
	//           Name:              createResp.SigningPayload.Domain.Name,
	//           Version:           createResp.SigningPayload.Domain.Version,
	//           ChainID:           int(createResp.SigningPayload.Domain.ChainID),
	//           VerifyingContract: createResp.SigningPayload.Domain.VerifyingContract,
	//       },
	//   })

	// Step 2 — Payer submits the signature
	sigResp, err := client.Payments.Sign(ctx, createResp.PaymentID, rail0.PayerSignatureRequest{
		V: 27, // from signature
		R: "0x1111111111111111111111111111111111111111111111111111111111111111",
		S: "0x2222222222222222222222222222222222222222222222222222222222222222",
	})
	if err != nil {
		log.Fatalf("Sign: %v", err)
	}
	fmt.Printf("Signature status: %s\n", sigResp.Status)

	// ----------------------------------------------------------------
	// Step 3 — Payee authorizes (relays the stored signature on-chain)
	// ----------------------------------------------------------------

	authResp, err := client.Payments.Authorize(ctx, createResp.PaymentID)
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Authorize failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Authorize: %v", err)
	}
	fmt.Printf("Authorized: tx=%s capturable=%s\n", authResp.TransactionHash, authResp.CapturableAmount)

	// ----------------------------------------------------------------
	// Step 4a — Payee prepares and submits a capture transaction
	// ----------------------------------------------------------------

	prepCapture, err := client.Payments.PrepareCapture(ctx, createResp.PaymentID, rail0.CapturePaymentRequest{
		Amount: "50000000",
	})
	if err != nil {
		log.Fatalf("PrepareCapture: %v", err)
	}

	// The payee signs prepCapture.UnsignedTransaction offline, then submits:
	//   signedTx := payeeWallet.SignTransaction(prepCapture.UnsignedTransaction)
	signedTx := "0x02f8..." // placeholder — use eth_signTransaction or a secp256k1 library

	captureResp, err := client.Payments.SubmitCapture(ctx, createResp.PaymentID, rail0.SubmitTransactionRequest{
		SignedTransaction: signedTx,
	})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("SubmitCapture failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("SubmitCapture: %v", err)
	}

	fmt.Printf("Captured: tx=%s captured=%s\n", captureResp.TransactionHash, captureResp.CapturedAmount)
	_ = prepCapture // suppress unused warning for the unsigned tx

	// ----------------------------------------------------------------
	// Step 4b — Alternatively: payee voids (order cancelled)
	// ----------------------------------------------------------------

	// prepVoid, _ := client.Payments.PrepareVoid(ctx, createResp.PaymentID)
	// signedVoidTx := payeeWallet.SignTransaction(prepVoid.UnsignedTransaction)
	// client.Payments.SubmitVoid(ctx, createResp.PaymentID, rail0.SubmitTransactionRequest{SignedTransaction: signedVoidTx})

	// ----------------------------------------------------------------
	// Step 4c — Release (fallback after authorizationExpiry, permissionless)
	// ----------------------------------------------------------------

	// releaseResp, _ := client.Payments.Release(ctx, createResp.PaymentID)
}
