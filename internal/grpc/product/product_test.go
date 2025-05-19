package productgrpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	product1 "github.com/KuranovNikita/ecomProto/gen/go/product"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockProductServer struct {
	product1.UnimplementedProductServiceServer

	GetProductFunc   func(ctx context.Context, req *product1.GetProductRequest) (*product1.GetProductResponse, error)
	ListProductsFunc func(ctx context.Context, req *product1.ListProductsRequest) (*product1.ListProductsResponse, error)
	CheckStockFunc   func(ctx context.Context, req *product1.CheckStockRequest) (*product1.CheckStockResponse, error)
	UpdateStockFunc  func(ctx context.Context, req *product1.UpdateStockRequest) (*emptypb.Empty, error)
}

func (s *mockProductServer) GetProduct(ctx context.Context, req *product1.GetProductRequest) (*product1.GetProductResponse, error) {
	if s.GetProductFunc != nil {
		return s.GetProductFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method GetProduct not implemented")
}

func (s *mockProductServer) ListProducts(ctx context.Context, req *product1.ListProductsRequest) (*product1.ListProductsResponse, error) {
	if s.ListProductsFunc != nil {
		return s.ListProductsFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method ListProducts not implemented")
}

func (s *mockProductServer) CheckStock(ctx context.Context, req *product1.CheckStockRequest) (*product1.CheckStockResponse, error) {
	if s.CheckStockFunc != nil {
		return s.CheckStockFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method CheckStock not implemented")
}

func (s *mockProductServer) UpdateStock(ctx context.Context, req *product1.UpdateStockRequest) (*emptypb.Empty, error) {
	if s.UpdateStockFunc != nil {
		return s.UpdateStockFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method UpdateStock not implemented")
}

func setupTestProductGRPCServer(t *testing.T, mockSrv *mockProductServer) (*Client, func()) {
	t.Helper()

	bufSize := 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcServer := grpc.NewServer()
	product1.RegisterProductServiceServer(grpcServer, mockSrv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	client, err := New(
		slog.Default(),
		"passthrough:///bufnet",
		1*time.Second,
		1,
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "Failed to create gRPC client for test")

	cleanup := func() {
		grpcServer.GracefulStop()
		lis.Close()
	}

	return client, cleanup
}

func TestClient_GetProduct_Success(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(101)
	expectedDetails := &product1.ProductDetails{
		Id:          productID,
		Name:        "Test Product",
		Description: "testing",
		Price:       1999,
		StockCount:  50,
	}

	mockSrv.GetProductFunc = func(ctx context.Context, req *product1.GetProductRequest) (*product1.GetProductResponse, error) {
		assert.Equal(t, productID, req.ProductId)
		return &product1.GetProductResponse{ProductDetails: expectedDetails}, nil
	}

	details, err := client.GetProduct(context.Background(), productID)

	assert.NoError(t, err)
	require.NotNil(t, details)
	assert.Equal(t, expectedDetails.Id, details.Id)
	assert.Equal(t, expectedDetails.Name, details.Name)
	assert.Equal(t, expectedDetails.Price, details.Price)
	assert.Equal(t, expectedDetails.StockCount, details.StockCount)
}

func TestClient_GetProduct_NotFound(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(999)
	serverError := status.Error(codes.NotFound, "product not found")

	mockSrv.GetProductFunc = func(ctx context.Context, req *product1.GetProductRequest) (*product1.GetProductResponse, error) {
		return nil, serverError
	}

	_, err := client.GetProduct(context.Background(), productID)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, err.Error(), "grpc.product.get_product")
}

func TestClient_ListProducts_Success(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	filter := "electronics"
	expectedProducts := []*product1.ProductDetails{
		{Id: 1, Name: "Laptop", Price: 120000, StockCount: 10},
		{Id: 2, Name: "Mouse", Price: 2500, StockCount: 100},
	}

	mockSrv.ListProductsFunc = func(ctx context.Context, req *product1.ListProductsRequest) (*product1.ListProductsResponse, error) {
		assert.Equal(t, filter, req.Filter)
		return &product1.ListProductsResponse{Products: expectedProducts}, nil
	}

	products, err := client.ListProducts(context.Background(), filter)

	assert.NoError(t, err)
	assert.NotNil(t, products)
	assert.Len(t, products, len(expectedProducts))
	for i := range expectedProducts {
		assert.True(t, proto.Equal(expectedProducts[i], products[i]), "product at index %d does not match", i)
	}
}

func TestClient_ListProducts_Empty(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	filter := "nonexistentcategory"
	var expectedProducts []*product1.ProductDetails // nil slice or empty slice

	mockSrv.ListProductsFunc = func(ctx context.Context, req *product1.ListProductsRequest) (*product1.ListProductsResponse, error) {
		return &product1.ListProductsResponse{Products: expectedProducts}, nil
	}

	products, err := client.ListProducts(context.Background(), filter)

	assert.NoError(t, err)
	if expectedProducts == nil {
		assert.Nil(t, products)
	} else {
		assert.NotNil(t, products)
		assert.Empty(t, products)
	}
}

func TestClient_CheckStock_Available(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(202)
	quantity := int32(15)

	mockSrv.CheckStockFunc = func(ctx context.Context, req *product1.CheckStockRequest) (*product1.CheckStockResponse, error) {
		assert.Equal(t, productID, req.ProductId)
		assert.Equal(t, quantity, req.Quantity)
		return &product1.CheckStockResponse{IsAvailable: true}, nil
	}

	isAvailable, err := client.CheckStock(context.Background(), productID, quantity)

	assert.NoError(t, err)
	assert.True(t, isAvailable)
}

func TestClient_CheckStock_NotAvailable(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(203)
	quantity := int32(100)

	mockSrv.CheckStockFunc = func(ctx context.Context, req *product1.CheckStockRequest) (*product1.CheckStockResponse, error) {
		assert.Equal(t, productID, req.ProductId)
		assert.Equal(t, quantity, req.Quantity)
		return &product1.CheckStockResponse{IsAvailable: false}, nil
	}

	isAvailable, err := client.CheckStock(context.Background(), productID, quantity)

	assert.NoError(t, err)
	assert.False(t, isAvailable)
}

func TestClient_CheckStock_ServerError(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	serverError := status.Error(codes.Internal, "stock database error")
	mockSrv.CheckStockFunc = func(ctx context.Context, req *product1.CheckStockRequest) (*product1.CheckStockResponse, error) {
		return nil, serverError
	}

	_, err := client.CheckStock(context.Background(), 204, 1)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, err.Error(), "grpc.product.check_stock")
}

func TestClient_UpdateStock_Success(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(301)
	quantityChange := int32(-2)

	mockSrv.UpdateStockFunc = func(ctx context.Context, req *product1.UpdateStockRequest) (*emptypb.Empty, error) {
		assert.Equal(t, productID, req.ProductId)
		assert.Equal(t, quantityChange, req.QuantityChange)
		return &emptypb.Empty{}, nil
	}

	err := client.UpdateStock(context.Background(), productID, quantityChange)

	assert.NoError(t, err)
}

func TestClient_UpdateStock_ProductNotFound(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(999)
	quantityChange := int32(-1)
	serverError := status.Error(codes.NotFound, "product not found for stock update")

	mockSrv.UpdateStockFunc = func(ctx context.Context, req *product1.UpdateStockRequest) (*emptypb.Empty, error) {
		return nil, serverError
	}

	err := client.UpdateStock(context.Background(), productID, quantityChange)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, err.Error(), "grpc.product.update_stock")
}

func TestClient_UpdateStock_InsufficientStock(t *testing.T) {
	mockSrv := &mockProductServer{}
	client, cleanup := setupTestProductGRPCServer(t, mockSrv)
	defer cleanup()

	productID := int64(302)
	quantityChange := int32(-10)
	serverError := status.Error(codes.FailedPrecondition, "insufficient stock to update")

	mockSrv.UpdateStockFunc = func(ctx context.Context, req *product1.UpdateStockRequest) (*emptypb.Empty, error) {
		return nil, serverError
	}

	err := client.UpdateStock(context.Background(), productID, quantityChange)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Contains(t, err.Error(), "grpc.product.update_stock")
}
