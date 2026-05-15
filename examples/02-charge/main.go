// One-shot payment: charge
//
// Combines authorize and capture in a single transaction — funds go
// directly from the buyer to the merchant with no escrow window.
// Use this when there is no need for a hold period (e.g. digital goods,
// instant fulfilment).
//
// On-chain flow:
//
//	buyer signs EIP-3009 → Charge()  funds move buyer → merchant (minus fee), atomically
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

	payment := rail0.Payment{
		Payer:               "0xBuyerAddress000000000000000000000000000000",
		Payee:               "0xMerchantAddress0000000000000000000000000000",
		Token:               "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
		MaxAmount:           "25000000",                                    // 25 USDC
		AuthorizationExpiry: now + 60*5,                                   // short window — charge captures immediately
		RefundExpiry:        now + 60*60*24*30,                            // 30-day refund window
		FeeBps:              0,
		FeeReceiver:         "0x0000000000000000000000000000000000000000",
	}

	const paymentID = "0xdeadbeef00000000000000000000000000000000000000000000000000000001"

	// Fetch the charge nonce (different from the authorize nonce).
	nonceResp, err := client.Payments.ChargeNonce(ctx, paymentID, payment.Payer)
	if err != nil {
		log.Fatalf("ChargeNonce: %v", err)
	}

	// The buyer signs transferWithAuthorization off-chain using SignCharge:
	//
	//   key, _ := rail0.HexToPrivateKey("0xYourPrivateKey")
	//   sig, _ := rail0.SignCharge(rail0.SignPaymentParams{
	//       PrivateKey:      key,
	//       Payment:         payment,
	//       Amount:          big.NewInt(25_000_000),
	//       Nonce:           nonceResp.Nonce,
	//       ContractAddress: "0xRAIL0ContractAddress",
	//       TokenDomain: rail0.TokenDomain{
	//           Name: "USD Coin", Version: "2", ChainID: 8453,
	//           VerifyingContract: payment.Token,
	//       },
	//   })

	tx, err := client.Payments.Charge(ctx, paymentID, rail0.ChargeParams{
		Payment: payment,
		Amount:  "25000000", // 25 USDC — exact amount, no hold
		V:       27,         // from signature
		R:       "0x1111111111111111111111111111111111111111111111111111111111111111",
		S:       "0x2222222222222222222222222222222222222222222222222222222222222222",
	})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Charge failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Charge: %v", err)
	}

	fmt.Printf("Charged: %s — status: %s\n", tx.TransactionHash, tx.Status)
	fmt.Printf("Nonce used: %s\n", nonceResp.Nonce)
}
