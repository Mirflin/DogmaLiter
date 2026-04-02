package payment

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"

	"backend/internal/auth"
	"backend/internal/config"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	cfg *config.Config
	jwt *auth.JWTManager
}

func NewHandler(db *gorm.DB, cfg *config.Config, jwt *auth.JWTManager) *Handler {
	stripe.Key = cfg.StripeSecretKey
	return &Handler{db: db, cfg: cfg, jwt: jwt}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware(h.jwt))
		r.Post("/checkout", h.CreateCheckout)
		r.Post("/cancel", h.CancelSubscription)
	})

	r.Post("/webhook", h.HandleWebhook)

	return r
}

func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	var body struct {
		PlanID       string `json:"plan_id"`
		BillingCycle string `json:"billing_cycle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	var priceID string
	switch body.PlanID {
	case "plus":
		priceID = h.cfg.StripePlusPriceID
	case "pro":
		priceID = h.cfg.StripeProPriceID
	default:
		respondJSON(w, 400, map[string]string{"error": "invalid plan"})
		return
	}

	if priceID == "" {
		respondJSON(w, 500, map[string]string{"error": "stripe price not configured"})
		return
	}

	var stripeCustomerID *string
	h.db.Raw("SELECT stripe_customer_id FROM users WHERE id = ?", userID).Scan(&stripeCustomerID)

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(h.cfg.FrontendURL + "/plans?success=true"),
		CancelURL:  stripe.String(h.cfg.FrontendURL + "/plans?canceled=true"),
		Metadata: map[string]string{
			"user_id": userID,
			"plan_id": body.PlanID,
		},
	}

	if stripeCustomerID != nil && *stripeCustomerID != "" {
		params.Customer = stripeCustomerID
	} else {
		var email string
		h.db.Raw("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
		params.CustomerEmail = stripe.String(email)
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("Stripe checkout error: %v", err)
		respondJSON(w, 500, map[string]string{"error": "failed to create checkout session"})
		return
	}

	respondJSON(w, 200, map[string]string{"url": s.URL})
}

func (h *Handler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	var subID *string
	h.db.Raw("SELECT stripe_subscription_id FROM users WHERE id = ?", userID).Scan(&subID)

	if subID == nil || *subID == "" {
		respondJSON(w, 400, map[string]string{"error": "no active subscription"})
		return
	}

	_, err := subscription.Cancel(*subID, nil)
	if err != nil {
		log.Printf("Stripe cancel error: %v", err)
		respondJSON(w, 500, map[string]string{"error": "failed to cancel subscription"})
		return
	}

	h.db.Exec("UPDATE users SET plan_id = 'free', stripe_subscription_id = NULL WHERE id = ?", userID)

	respondJSON(w, 200, map[string]string{"message": "subscription canceled"})
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		w.WriteHeader(400)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.cfg.StripeWebhookKey)
	if err != nil {
		log.Printf("Stripe webhook signature error: %v", err)
		w.WriteHeader(400)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Printf("Webhook parse error: %v", err)
			w.WriteHeader(400)
			return
		}

		userID := sess.Metadata["user_id"]
		planID := sess.Metadata["plan_id"]

		if userID == "" || planID == "" {
			log.Printf("Webhook missing metadata")
			w.WriteHeader(400)
			return
		}

		customerID := ""
		if sess.Customer != nil {
			customerID = sess.Customer.ID
		}
		subscriptionID := ""
		if sess.Subscription != nil {
			subscriptionID = sess.Subscription.ID
		}

		h.db.Exec(
			"UPDATE users SET plan_id = ?, stripe_customer_id = ?, stripe_subscription_id = ? WHERE id = ?",
			planID, customerID, subscriptionID, userID,
		)
		log.Printf("User %s upgraded to plan %s", userID, planID)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("Webhook parse error: %v", err)
			w.WriteHeader(400)
			return
		}

		h.db.Exec(
			"UPDATE users SET plan_id = 'free', stripe_subscription_id = NULL WHERE stripe_subscription_id = ?",
			sub.ID,
		)
		log.Printf("Subscription %s canceled, user downgraded to free", sub.ID)
	}

	w.WriteHeader(200)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
