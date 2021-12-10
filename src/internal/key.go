package internal

import (
	"os"
)

func StripeSecretKey() string {
	return os.Getenv("STRIPE_SECRET_KEY")
}
