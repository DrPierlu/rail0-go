package rail0

import "fmt"

// APIError is returned when the RAIL0 API responds with a non-2xx status code.
type APIError struct {
	// Status is the HTTP status code (e.g. 404, 409, 422).
	Status int
	// Code is the machine-readable error identifier, e.g. "PaymentNotFound".
	Code string
	// Message is the human-readable description.
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("rail0 %s (HTTP %d): %s", e.Code, e.Status, e.Message)
}
