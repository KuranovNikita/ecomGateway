package processor

import (
	"context"
	ordergrpc "ecomGateway/internal/grpc/order"
	productgrpc "ecomGateway/internal/grpc/product"
	usergrpc "ecomGateway/internal/grpc/user"
	"fmt"
	"log"
)

type Processor interface {
	RegisterUser(ctx context.Context, email, password, login string) (int64, error)
	LoginUser(ctx context.Context, login, password string) (string, error)
	// ListProducts(ctx context.Context, filter, id string) ([]Product, error)
	// CreateOrder(ctx context.Context, userID int64, items []OrderItemHTTP) (*Order, error)
	// ListUserOrders(ctx context.Context, userID int64) ([]OrderDTO, error)
}

type processorService struct {
	userClient    usergrpc.Client
	orderClient   ordergrpc.Client
	productClient productgrpc.Client
}

type Product struct {
	Id          int64
	Name        string
	Description string
	Price       int64
	StockCount  int32
}

func NewProcessorService(
	userClient usergrpc.Client,
	orderClient ordergrpc.Client,
	productClient productgrpc.Client,
) Processor {
	return &processorService{
		userClient:    userClient,
		productClient: productClient,
		orderClient:   orderClient,
	}
}

func (s *processorService) RegisterUser(ctx context.Context, email, password, login string) (int64, error) {
	resp, err := s.userClient.Register(ctx, email, login, password)

	if err != nil {
		log.Printf("Error registering user: %v", err)
		return 0, fmt.Errorf("user service error: %w", err)
	}
	return resp, nil
}

func (s *processorService) LoginUser(ctx context.Context, login, password string) (string, error) {
	resp, err := s.userClient.Login(ctx, login, password)
	if err != nil {
		log.Printf("Error logging user : %v", err)
		return "", fmt.Errorf("user service error: %w", err)
	}

	return resp, nil
}

// func (s *processorService) ListProducts(ctx context.Context, filter, id string) ([]Product, error) {

// }
