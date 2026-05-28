// One-shot payment: charge
//
// Combines authorize and capture in a single transaction — funds go
// directly from the payer to the payee with no escrow window.
// Use this when there is no need for a hold period (e.g. digital goods,
// instant fulfilment).
//
// On-chain flow:
//
//	payer signs EIP-712 → Charge + Submit  funds move payer → payee (minus fee), atomically
//
// Run:
//
//	go run examples/02-charge/main.go
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
	// Step 1 — Payer creates a payment intent (mode = "charge")
	// ----------------------------------------------------------------

	createResp, err := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
		Payment: rail0.PaymentInput{
			Payer:  "0xBuyerAddress000000000000000000000000000000",
			Payee:  "0xMerchantAddress0000000000000000000000000000",
			Token:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
			Amount: "25000000",                                    // 25 USDC (6 decimals)
		},
		ChainId: 8453, // Base
		Mode:    "charge",
	})
	if err != nil {
		log.Fatalf("CreatePayment: %v", err)
	}

	fmt.Printf("Payment ID: %s\n", createResp.PaymentId)

	// The payer signs the signingPayload using eth_signTypedData_v4 or SignCharge:
	//
	//   key, _ := rail0.HexToPrivateKey("0xYourPrivateKey")
	//   sig, _ := rail0.SignCharge(rail0.SignPaymentParams{
	//       PrivateKey:      key,
	//       Payment:         createResp.Payment,
	//       Amount:          big.NewInt(25_000_000),
	//       Nonce:           createResp.SigningPayload.Message.Nonce,
	//       ContractAddress: createResp.Rail0Contract,
	//       TokenDomain: rail0.TokenDomain{
	//           Name:              createResp.SigningPayload.Domain.Name,
	//           Version:           createResp.SigningPayload.Domain.Version,
	//           ChainID:           uint64(createResp.SigningPayload.Domain.ChainId),
	//           VerifyingContract: createResp.SigningPayload.Domain.VerifyingContract,
	//       },
	//   })
	//   signature := "0x" + sig.R[2:] + sig.S[2:] + fmt.Sprintf("%02x", sig.V)

	// Step 2 — Payer submits the 65-byte combined signature
	_, err = client.Payments.Sign(ctx, createResp.PaymentId, rail0.PayerSignatureRequest{
		Signature: "0x1a2b3c...(130 hex chars from eth_signTypedData_v4)",
	})
	if err != nil {
		log.Fatalf("Sign: %v", err)
	}

	// ----------------------------------------------------------------
	// Step 3 — Payee builds the charge transaction
	// ----------------------------------------------------------------

	prepCharge, err := client.Payments.Charge(ctx, createResp.PaymentId)
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Charge failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Charge: %v", err)
	}
	fmt.Printf("Unsigned charge tx: %s\n", prepCharge.UnsignedTransaction)

	// The payee signs prepCharge.UnsignedTransaction offline, then submits:
	//   signedChargeTx := payeeWallet.SignTransaction(prepCharge.UnsignedTransaction)
	signedChargeTx := "0x02f8..." // placeholder

	// Submit returns 202 immediately with status "submitting".
	// Poll Payments.Get until status advances to "charged".
	chargeSubmit, err := client.Payments.Submit(ctx, createResp.PaymentId,
		rail0.SubmitTransactionRequest{SignedTransaction: signedChargeTx})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Submit (charge) failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Submit (charge): %v", err)
	}

	fmt.Printf("Charge enqueued: id=%s status=%s\n", chargeSubmit.Rail0ID, chargeSubmit.Status)
	// poll until status == "charged": client.Payments.Get(ctx, createResp.PaymentId)
}
