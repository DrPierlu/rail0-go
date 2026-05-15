// Refund flow
//
// After a capture (or charge) the merchant can refund up to the full
// captured amount back to the buyer, as long as refundExpiry has not
// passed and the refundableAmount is sufficient.
//
// The refund is initiated by the payee (merchant). The API submits the
// transaction on behalf of the payee.
//
// On-chain flow:
//
//	merchant → Refund()  funds move merchant → buyer
//
// Run:
//
//	go run examples/03-refund/main.go
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
		Token:               "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		MaxAmount:           "100000000",
		AuthorizationExpiry: now - 30*60,       // already captured
		RefundExpiry:        now + 60*60*24*6,  // still within refund window
		FeeBps:              50,
		FeeReceiver:         "0xFeeReceiverAddress000000000000000000000000",
	}

	const paymentID = "0xdeadbeef00000000000000000000000000000000000000000000000000000002"

	// ----------------------------------------------------------------
	// Check current refundable balance before acting
	// ----------------------------------------------------------------

	state, err := client.Payments.Get(ctx, paymentID)
	if err != nil {
		log.Fatalf("Get: %v", err)
	}
	fmt.Printf("Refundable balance: %s\n", state.State.RefundableAmount) // e.g. "50000000"

	// ----------------------------------------------------------------
	// Refund — partial or full
	// ----------------------------------------------------------------

	tx, err := client.Payments.Refund(ctx, paymentID, rail0.RefundParams{
		Payment: payment,
		Amount:  "50000000", // partial refund — 50 USDC out of 50 captured
	})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			// Common: RefundExpired, InvalidRefundAmount, NotPayee
			log.Fatalf("Refund failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("Refund: %v", err)
	}

	fmt.Printf("Refunded: %s\n", tx.TransactionHash)
}
