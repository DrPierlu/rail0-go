// Refund flow
//
// After a capture (or charge) the merchant can refund up to the full
// captured amount back to the payer, as long as refundExpiry has not
// passed. The payee must first approve the RAIL0 contract as a spender
// on the token (so the contract can pull funds back from the payee).
//
// On-chain flow:
//
//	payee signs approve tx → approve()  RAIL0 contract approved as spender
//	payee signs refund tx  → Refund()   funds move payee → payer
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

	rail0 "github.com/rail0/go-sdk"
)

func main() {
	ctx := context.Background()
	client := rail0.NewClient(rail0.ClientOptions{
		BaseURL: "https://api.rail0.xyz",
	})

	// Assume the payment was previously created and captured.
	const paymentID = "0xdeadbeef00000000000000000000000000000000000000000000000000000002"

	// ----------------------------------------------------------------
	// Step 1 — Payee approves the RAIL0 contract as token spender
	// (required before the contract can pull funds back from the payee)
	// ----------------------------------------------------------------

	prepApprove, err := client.Payments.PrepareApprove(ctx, paymentID, rail0.ApproveRequest{
		Amount: "115792089237316195423570985008687907853269984665640564039457584007913129639935", // unlimited
	})
	if err != nil {
		log.Fatalf("PrepareApprove: %v", err)
	}

	// Payee signs prepApprove.UnsignedTransaction offline, then submits:
	//   signedApproveTx := payeeWallet.SignTransaction(prepApprove.UnsignedTransaction)
	signedApproveTx := "0x02f8..." // placeholder

	approveResp, err := client.Payments.SubmitApprove(ctx, paymentID, rail0.SubmitApproveRequest{
		SignedTransaction: signedApproveTx,
	})
	if err != nil {
		log.Fatalf("SubmitApprove: %v", err)
	}
	fmt.Printf("Approved: tx=%s spender=%s\n", approveResp.TransactionHash, approveResp.Spender)
	_ = prepApprove

	// ----------------------------------------------------------------
	// Step 2 — Payee prepares and submits a refund transaction
	// ----------------------------------------------------------------

	prepRefund, err := client.Payments.PrepareRefund(ctx, paymentID, rail0.RefundPaymentRequest{
		Amount: "50000000", // partial or full refund
	})
	if err != nil {
		log.Fatalf("PrepareRefund: %v", err)
	}

	// Payee signs prepRefund.UnsignedTransaction offline, then submits:
	//   signedRefundTx := payeeWallet.SignTransaction(prepRefund.UnsignedTransaction)
	signedRefundTx := "0x02f8..." // placeholder

	refundResp, err := client.Payments.SubmitRefund(ctx, paymentID, rail0.SubmitTransactionRequest{
		SignedTransaction: signedRefundTx,
	})
	if err != nil {
		var apiErr *rail0.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("SubmitRefund failed [%s]: %s", apiErr.Code, apiErr.Message)
		}
		log.Fatalf("SubmitRefund: %v", err)
	}

	fmt.Printf("Refunded: tx=%s refunded=%s remaining=%s\n",
		refundResp.TransactionHash, refundResp.RefundedAmount, refundResp.RefundableAmount)
	_ = prepRefund
}
