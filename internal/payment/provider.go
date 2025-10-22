package payment

type PaymentProvider interface {
	CreateCheckoutSession(params interface{}) (interface{}, error)
	VerifyWebhookSignature(payload []byte, signature string) (interface{}, error)
}

type PaymentService struct {
	Stripe   *StripeProvider
	Paystack *PaystackProvider
}

func NewPaymentService(stripeKey, stripeWebhook, paystackKey, paystackWebhook string) *PaymentService {
	return &PaymentService{
		Stripe:   NewStripeProvider(stripeKey, stripeWebhook),
		Paystack: NewPaystackProvider(paystackKey, paystackWebhook),
	}
}