package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/innomon/aigen-app/core/services"
)

type CommerceApi struct {
	commerceService *services.CommerceService
	authApi         *AuthApi
}

func NewCommerceApi(commerceService *services.CommerceService, authApi *AuthApi) *CommerceApi {
	return &CommerceApi{
		commerceService: commerceService,
		authApi:         authApi,
	}
}

func (a *CommerceApi) Register(r chi.Router) {
	r.Route("/api/commerce", func(r chi.Router) {
		r.Get("/search", a.SearchProducts)
		r.Post("/checkout", a.CreateCheckout)
		r.Get("/verify/{mandateId}", a.VerifyMandate)
	})
}

func (a *CommerceApi) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	products, err := a.commerceService.SearchProducts(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(products)
}

type checkoutRequest struct {
	BuyerID    string   `json:"buyer_id"`
	ProductIDs []string `json:"product_ids"`
}

func (a *CommerceApi) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	checkout, err := a.commerceService.CreateCheckout(r.Context(), req.BuyerID, req.ProductIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(checkout)
}

func (a *CommerceApi) VerifyMandate(w http.ResponseWriter, r *http.Request) {
	mandateId := chi.URLParam(r, "mandateId")
	verified, err := a.commerceService.VerifyMandate(r.Context(), mandateId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"verified": verified})
}
