package services

import (
	"context"
	"fmt"

	"github.com/innomon/aigen-app/utils/datamodels"
)

type CommerceService struct {
	entityService IEntityService
}

func NewCommerceService(entityService IEntityService) *CommerceService {
	return &CommerceService{
		entityService: entityService,
	}
}

func (s *CommerceService) SearchProducts(ctx context.Context, query string) ([]datamodels.Record, error) {
	filters := []datamodels.Filter{}
	if query != "" {
		filters = append(filters, datamodels.Filter{
			FieldName: "name",
			MatchType: "contains",
			Constraints: []datamodels.Constraint{
				{Match: "contains", Values: []interface{}{query}},
			},
		})
	}

	limit := "10"
	records, _, err := s.entityService.List(ctx, "ucp_product", datamodels.Pagination{Limit: &limit}, filters, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}
	return records, nil
}

func (s *CommerceService) CreateCheckout(ctx context.Context, buyerId string, productIds []string) (datamodels.Record, error) {
	// Simple implementation for now: calculate total and create checkout record
	var total float64
	var currency string

	for _, pid := range productIds {
		prod, err := s.entityService.Single(ctx, "ucp_product", pid)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch product %s: %w", pid, err)
		}
		
		// Handle potential nil or wrong type gracefully
		if p, ok := prod["price"].(float64); ok {
			total += p
		} else if p, ok := prod["price"].(int64); ok {
			total += float64(p)
		} else if p, ok := prod["price"].(float32); ok {
			total += float64(p)
		}

		if currency == "" {
			if c, ok := prod["currency"].(string); ok {
				currency = c
			}
		}
	}

	checkout := datamodels.Record{
		"buyer_id": buyerId,
		"total":    total,
		"currency": currency,
		"status":   "active",
	}

	saved, err := s.entityService.Insert(ctx, "ucp_checkout", checkout)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout: %w", err)
	}

	return saved, nil
}

func (s *CommerceService) VerifyMandate(ctx context.Context, mandateId string) (bool, error) {
	// Placeholder for AP2 verification logic
	mandate, err := s.entityService.Single(ctx, "ap2_mandate", mandateId)
	if err != nil {
		return false, fmt.Errorf("failed to fetch mandate: %w", err)
	}

	// In a real implementation, we would use the ap2 package from ucp-srv
	// For now, we assume it's valid if it exists
	return mandate != nil, nil
}
