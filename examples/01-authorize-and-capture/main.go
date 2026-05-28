// Standard two-step payment flow: authorize → capture
//
// The payer creates a payment intent, signs the EIP-712 payload, and
// submits the signature. The payee then builds the authorize transaction,
// signs it offline, and broadcasts it (funds move to escrow). Finally
// the payee builds a capture transaction, signs it, and broadcasts it.
//
// On-chain flow:
//
//	payer signs EIP-712    → Authorize + Submit   funds move payer → escrow
//	payee signs capture tx → PrepareCapture + Submit funds move escrow → payee (minus fee)
//	payee signs void tx    → PrepareVoid + Submit    alternative: funds move escrow → payer
//	anyone                 → PrepareRelease + Submit  fallback after authorizationExpiry
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

	rail0 "github.com/rail0/go-sdk"
)

func main() {
	ctx := context.Background()
	client := rail0.NewClient(rail0.ClientOptions{
		BaseURL: "https://api.rail0.xyz",
	})

	// ----------------------------------------------------------------
	// Step 1 — Payer creates a payment intent
	// ----------------------------------------------------------------

	createResp, err := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
		Payment: rail0.PaymentInput{
			Payer:  "0xBuyerAddress000000000000000000000000000000",
			Payee:  "0xMerchantAddress0000000000000000000000000000",
			Token:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
			Amount: "50000000",                                    // 50 USDC (6 decimals)
		},
		ChainId: 8453, // Base
		Mode:    "authorize",
	})
	if err != nil {
		log.Fatalf("CreatePayment: %v", err)
	}

	fmt.Printf("Payment ID:  %s\n", createResp.PaymentId)
	fmt.Printf("Config hash: %s\n", createResp.ConfigHash)

	// The payer signs the signingPayload using eth_signTypedData_v4 (wallet)
	// or with SignAuthorize (direct key access):
	//
	//   key, _ := rail0.HexToPrivateKey("0xYourPrivateKey")
	//   sig, _ := rail0.SignAuthorize(rail0.SignPaymentParams{
	//       PrivateKey:      key,
	//       Payment:         createResp.Payment,
	//       Amount:          big.NewInt(50_000_000),
	//       Nonce:           createResp.SigningPayload.Message.Nonce,
	//       ContractAddress: createResp.Rail0Contract,
	//       TokenDomain: rail0.TokenDomain{
	//           Name:              createResp.SigningPayload.Domain.Name,
	//           Version:           createResp.SigningPayload.Domain.Version,
	//           ChainID:           uint64(createResp.SigningPayload.Domain.ChainId),
	//           VerifyingContract: createResp.SigningPayload.Domain.VerifyingContract,
	//       },
	//   })
	//   // sig is an Eip3009Signature{V, R, S}; combine into one hex string:
	//   signature := sig.R[2:] + sig.S[2:] + fmt.Sprintf("%02x", sig.V)
	//   signature = "0x" + signature

	// Step 2 — Payer submits the 65-byte combined signature
	sigResp, err := client.Payments.Sign(ctx, createResp.PaymentId, rail0.PayerSignatureRequest{
		Signature: "0x1a2b3c...(130 hex chars from eth_signTypedData_v4)",
	})
	if err != nil {
		log.Fatalf("Sign: %v", err)
	}
	fmt.Printf("Signature status: %s\n", sigResp.Status)

	// ----------------------------------------------------------------
	// Step 3 — Payee builds and broadcasts the authorize transaction
	// ----------------------------------------------------------------

	prepAuth, err := client.Payments.Authorize(ctx, createResp.PaymentId)
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Authorize failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Authorize: %v", err)
	}
	fmt.Printf("Unsigned authorize tx: %s\n", prepAuth.UnsignedTransaction)

	// The payee signs prepAuth.UnsignedTransaction offline, then submits:
	//   signedAuthTx := payeeWallet.SignTransaction(prepAuth.UnsignedTransaction)
	signedAuthTx := "0x02f8..." // placeholder — use eth_signTransaction or secp256k1 library

	// Submit returns 202 immediately with status "submitting".
	// Poll Payments.Get until status advances to "authorized".
	authSubmit, err := client.Payments.Submit(ctx, createResp.PaymentId,
		rail0.SubmitTransactionRequest{SignedTransaction: signedAuthTx})
	if err != nil {
		log.Fatalf("Submit (authorize): %v", err)
	}
	fmt.Printf("Authorize enqueued: id=%s status=%s\n", authSubmit.Rail0ID, authSubmit.Status)
	// poll until status == "authorized": client.Payments.Get(ctx, createResp.PaymentId)

	// ----------------------------------------------------------------
	// Step 4a — Payee prepares and submits a capture transaction
	// ----------------------------------------------------------------

	prepCapture, err := client.Payments.PrepareCapture(ctx, createResp.PaymentId,
		rail0.CapturePaymentRequest{Amount: "50000000"})
	if err != nil {
		log.Fatalf("PrepareCapture: %v", err)
	}

	// The payee signs prepCapture.UnsignedTransaction offline, then submits:
	//   signedCaptureTx := payeeWallet.SignTransaction(prepCapture.UnsignedTransaction)
	signedCaptureTx := "0x02f8..." // placeholder
	_ = prepCapture

	// Submit returns 202 immediately. Poll until status == "captured" or "partially_captured".
	captureSubmit, err := client.Payments.Submit(ctx, createResp.PaymentId,
		rail0.SubmitTransactionRequest{SignedTransaction: signedCaptureTx})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Submit (capture) failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Submit (capture): %v", err)
	}
	fmt.Printf("Capture enqueued: id=%s status=%s\n", captureSubmit.Rail0ID, captureSubmit.Status)

	// ----------------------------------------------------------------
	// Step 4b — Alternatively: payee voids (order cancelled)
	// ----------------------------------------------------------------

	// prepVoid, _ := client.Payments.PrepareVoid(ctx, createResp.PaymentId)
	// signedVoidTx := payeeWallet.SignTransaction(prepVoid.UnsignedTransaction)
	// client.Payments.Submit(ctx, createResp.PaymentId, rail0.SubmitTransactionRequest{SignedTransaction: signedVoidTx})

	// ----------------------------------------------------------------
	// Step 4c — Release (fallback after authorizationExpiry, permissionless)
	// ----------------------------------------------------------------

	// prepRelease, _ := client.Payments.PrepareRelease(ctx, createResp.PaymentId, rail0.ReleaseRequest{})
	// signedReleaseTx := payeeWallet.SignTransaction(prepRelease.UnsignedTransaction)
	// client.Payments.Submit(ctx, createResp.PaymentId, rail0.SubmitTransactionRequest{SignedTransaction: signedReleaseTx})
}
