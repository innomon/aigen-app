package services

import (
	"context"
	"testing"

	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEntityService for testing
type MockEntityService struct {
	mock.Mock
}

func (m *MockEntityService) List(ctx context.Context, name string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	args := m.Called(ctx, name, pagination, filters, sorts)
	return args.Get(0).([]datamodels.Record), args.Get(1).(int64), args.Error(2)
}

func (m *MockEntityService) Single(ctx context.Context, name string, id interface{}) (datamodels.Record, error) {
	args := m.Called(ctx, name, id)
	return args.Get(0).(datamodels.Record), args.Error(1)
}

func (m *MockEntityService) Insert(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	args := m.Called(ctx, name, data)
	return args.Get(0).(datamodels.Record), args.Error(1)
}

func (m *MockEntityService) Update(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error) {
	args := m.Called(ctx, name, data)
	return args.Get(0).(datamodels.Record), args.Error(1)
}

func (m *MockEntityService) Delete(ctx context.Context, name string, id interface{}) error {
	args := m.Called(ctx, name, id)
	return args.Error(0)
}

func (m *MockEntityService) CollectionList(ctx context.Context, name, id, attr string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	return nil, 0, nil
}

func (m *MockEntityService) CollectionInsert(ctx context.Context, name, id, attr string, data datamodels.Record) (datamodels.Record, error) {
	return nil, nil
}

func (m *MockEntityService) JunctionList(ctx context.Context, name, id, attr string, exclude bool, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error) {
	return nil, 0, nil
}

func (m *MockEntityService) JunctionSave(ctx context.Context, name, id, attr string, targetIds []interface{}) error {
	return nil
}

func (m *MockEntityService) JunctionDelete(ctx context.Context, name, id, attr string, targetIds []interface{}) error {
	return nil
}

func TestCommerceService_SearchProducts(t *testing.T) {
	mockEntity := new(MockEntityService)
	svc := NewCommerceService(mockEntity)

	ctx := context.Background()
	query := "token"
	limit := "10"

	mockEntity.On("List", ctx, "ucp_product", datamodels.Pagination{Limit: &limit}, mock.Anything, mock.Anything).
		Return([]datamodels.Record{
			{"id": 1, "name": "AiGen Token", "price": 19.99},
		}, int64(1), nil)

	products, err := svc.SearchProducts(ctx, query)

	assert.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "AiGen Token", products[0]["name"])
	mockEntity.AssertExpectations(t)
}

func TestCommerceService_CreateCheckout(t *testing.T) {
	mockEntity := new(MockEntityService)
	svc := NewCommerceService(mockEntity)

	ctx := context.Background()
	buyerId := "user_1"
	productIds := []string{"1"}

	mockEntity.On("Single", ctx, "ucp_product", "1").
		Return(datamodels.Record{"id": 1, "name": "AiGen Token", "price": 19.99, "currency": "USD"}, nil)

	mockEntity.On("Insert", ctx, "ucp_checkout", mock.MatchedBy(func(r datamodels.Record) bool {
		return r["buyer_id"] == buyerId && r["total"] == 19.99
	})).Return(datamodels.Record{"id": 101, "buyer_id": buyerId, "total": 19.99, "status": "active"}, nil)

	checkout, err := svc.CreateCheckout(ctx, buyerId, productIds)

	assert.NoError(t, err)
	assert.Equal(t, 19.99, checkout["total"])
	assert.Equal(t, "active", checkout["status"])
	mockEntity.AssertExpectations(t)
}
