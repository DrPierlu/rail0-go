// Refund flow using EIP-3009 receiveWithAuthorization.
// No separate ERC-20 approve() needed.

package main

import (
	"context"
	"fmt"
	"log"

	rail0 "github.com/rail0/go-sdk"
)

func main() {
	ctx := context.Background()
	client := rail0.NewClient(rail0.ClientOptions{BaseURL: "https://api.rail0.xyz"})

	paymentID := "0x..." // a captured payment

	// ── Phase 1: get the EIP-3009 signing payload ──────────────────────────────
	phase1, err := client.Payments.RefundPrepare(ctx, paymentID, rail0.RefundPayloadRequest{
		Amount: "50000000",
	})
	if err != nil {
		log.Fatalf("RefundPayload phase 1: %v", err)
	}
	// phase1.SigningPayload contains the EIP-712 typed-data the payee must sign
	// using eth_signTypedData_v4 or rail0.SignRefund (when available).
	fmt.Printf("Signing payload nonce: %s\n", phase1.SigningPayload.Message.Nonce)

	// Payee signs the payload → v, r, s
	v, r, s := 27, "0xabc...", "0xdef..."

	// ── Phase 2: build the unsigned refund() tx with EIP-3009 sig embedded ────
	phase2, err := client.Payments.RefundPrepare(ctx, paymentID, rail0.RefundPayloadRequest{
		Amount: "50000000",
		V:      v,
		R:      r,
		S:      s,
	})
	if err != nil {
		log.Fatalf("RefundPayload phase 2: %v", err)
	}
	fmt.Printf("Unsigned refund tx: %s\n", phase2.UnsignedTransaction)

	// Payee signs the EIP-1559 tx → SIGNED_TX
	signedTx := "0x02f8..." // placeholder

	// ── Submit ────────────────────────────────────────────────────────────────
	refundSubmit, err := client.Payments.Refund(ctx, paymentID,
		rail0.SubmitTransactionRequest{SignedTransaction: signedTx})
	if err != nil {
		log.Fatalf("Refund: %v", err)
	}
	fmt.Printf("Refund enqueued: id=%s status=%s\n", refundSubmit.Rail0Id, refundSubmit.Status)
	// poll until status == "refunded": client.Payments.Get(ctx, paymentID)
}
