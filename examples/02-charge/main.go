// One-shot payment: charge
//
// Combines authorize and capture in a single transaction — funds go
// directly from the payer to the payee with no escrow window.
// Use this when there is no need for a hold period (e.g. digital goods,
// instant fulfilment).
//
// On-chain flow:
//
//	payer signs EIP-712 → Charge()  funds move payer → payee (minus fee), atomically
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
		MaxAmount:           "25000000",                                    // 25 USDC
		AuthorizationExpiry: now + 60*5,                                   // short window — charge captures immediately
		RefundExpiry:        now + 60*60*24*30,                            // 30-day refund window
		FeeBps:              0,
		FeeReceiver:         "0x0000000000000000000000000000000000000000",
	}

	// ----------------------------------------------------------------
	// Step 1 — Payer creates a payment intent (mode = "charge")
	// ----------------------------------------------------------------

	createResp, err := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{
		Payment: payment,
		Amount:  "25000000",
		ChainID: 8453,
		Mode:    "charge",
	})
	if err != nil {
		log.Fatalf("CreatePayment: %v", err)
	}

	fmt.Printf("Payment ID: %s\n", createResp.PaymentID)

	// The payer signs the signingPayload using eth_signTypedData_v4 or SignCharge:
	//
	//   key, _ := rail0.HexToPrivateKey("0xYourPrivateKey")
	//   sig := rail0.SignCharge(rail0.SignPaymentParams{
	//       PrivateKey:      key,
	//       Payment:         payment,
	//       Amount:          big.NewInt(25_000_000),
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
	_, err = client.Payments.Sign(ctx, createResp.PaymentID, rail0.PayerSignatureRequest{
		V: 27, // from signature
		R: "0x1111111111111111111111111111111111111111111111111111111111111111",
		S: "0x2222222222222222222222222222222222222222222222222222222222222222",
	})
	if err != nil {
		log.Fatalf("Sign: %v", err)
	}

	// ----------------------------------------------------------------
	// Step 3 — Payee triggers the one-shot charge
	// ----------------------------------------------------------------

	tx, err := client.Payments.Charge(ctx, createResp.PaymentID)
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Charge failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Charge: %v", err)
	}

	fmt.Printf("Charged: tx=%s charged=%s fee=%s\n",
		tx.TransactionHash, tx.ChargedAmount, tx.FeeAmount)
}
