package ordergrpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	order1 "github.com/KuranovNikita/ecomProto/gen/go/order"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/testing/protocmp"
)

type mockOrderServer struct {
	order1.UnimplementedOrderServiceServer

	CreateOrderFunc    func(ctx context.Context, req *order1.CreateOrderRequest) (*order1.CreateOrderResponse, error)
	GetOrderFunc       func(ctx context.Context, req *order1.GetOrderRequest) (*order1.GetOrderResponse, error)
	ListUserOrdersFunc func(ctx context.Context, req *order1.ListUserOrdersRequest) (*order1.ListUserOrdersResponse, error)
}

func (s *mockOrderServer) CreateOrder(ctx context.Context, req *order1.CreateOrderRequest) (*order1.CreateOrderResponse, error) {
	if s.CreateOrderFunc != nil {
		return s.CreateOrderFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method CreateOrder not implemented")
}

func (s *mockOrderServer) GetOrder(ctx context.Context, req *order1.GetOrderRequest) (*order1.GetOrderResponse, error) {
	if s.GetOrderFunc != nil {
		return s.GetOrderFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method GetOrder not implemented")
}

func (s *mockOrderServer) ListUserOrders(ctx context.Context, req *order1.ListUserOrdersRequest) (*order1.ListUserOrdersResponse, error) {
	if s.ListUserOrdersFunc != nil {
		return s.ListUserOrdersFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method ListUserOrders not implemented")
}

func setupTestOrderGRPCServer(t *testing.T, mockSrv *mockOrderServer) (*Client, func()) {
	t.Helper()

	bufSize := 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcServer := grpc.NewServer()
	order1.RegisterOrderServiceServer(grpcServer, mockSrv)

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

func TestNewOrderItem(t *testing.T) {
	productID := int64(101)
	quantity := int32(2)
	price := int64(1500)

	item := NewOrderItem(productID, quantity, price)

	assert.NotNil(t, item)
	assert.Equal(t, productID, item.ProductId)
	assert.Equal(t, quantity, item.Quantity)
	assert.Equal(t, price, item.Price)
}

func TestClient_CreateOrder_Success(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(1)
	items := []*order1.OrderItem{
		NewOrderItem(10, 1, 100),
		NewOrderItem(20, 2, 200),
	}
	expectedOrderID := int64(555)
	expectedTotalPrice := int64(500) // 1*100 + 2*200

	mockSrv.CreateOrderFunc = func(ctx context.Context, req *order1.CreateOrderRequest) (*order1.CreateOrderResponse, error) {
		assert.Equal(t, userID, req.UserId)

		sortOpt := cmpopts.SortSlices(func(a, b *order1.OrderItem) bool {
			return a.GetProductId() < b.GetProductId()
		})

		diff := cmp.Diff(items, req.GetItems(), protocmp.Transform(), sortOpt)
		if diff != "" {
			t.Errorf("CreateOrderRequest.Items mismatch (-want +got):\n%s", diff)
		}

		return &order1.CreateOrderResponse{OrderId: expectedOrderID, TotalPrice: expectedTotalPrice}, nil
	}

	orderID, totalPrice, err := client.CreateOrder(context.Background(), userID, items)

	assert.NoError(t, err)
	assert.Equal(t, expectedOrderID, orderID)
	assert.Equal(t, expectedTotalPrice, totalPrice)
}

func TestClient_CreateOrder_ServerError(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	serverError := status.Error(codes.Internal, "payment processing failed")
	mockSrv.CreateOrderFunc = func(ctx context.Context, req *order1.CreateOrderRequest) (*order1.CreateOrderResponse, error) {
		return nil, serverError
	}

	_, _, err := client.CreateOrder(context.Background(), 1, []*order1.OrderItem{{ProductId: 1, Quantity: 1, Price: 10}})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "payment processing failed")
	assert.Contains(t, err.Error(), "grpc.order.create_order")
}

func TestClient_GetOrder_Success(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	orderID := int64(789)

	expectedOrderDetails := &order1.OrderDetails{
		OrderId:    orderID,
		UserId:     int64(12),
		TotalPrice: int64(2500),
		Items: []*order1.OrderItem{
			{ProductId: 1, Quantity: 1, Price: 1000},
			{ProductId: 2, Quantity: 3, Price: 500},
		},
	}

	mockSrv.GetOrderFunc = func(ctx context.Context, req *order1.GetOrderRequest) (*order1.GetOrderResponse, error) {
		assert.Equal(t, orderID, req.OrderId)
		return &order1.GetOrderResponse{OrderDetails: &order1.OrderDetails{
			OrderId:    expectedOrderDetails.OrderId,
			UserId:     expectedOrderDetails.UserId,
			TotalPrice: expectedOrderDetails.TotalPrice,
			Items:      expectedOrderDetails.Items,
			Status:     expectedOrderDetails.Status,
		}}, nil
	}

	orderDetails, err := client.GetOrder(context.Background(), orderID)

	assert.NoError(t, err)
	assert.NotNil(t, orderDetails)

	assert.Equal(t, expectedOrderDetails.OrderId, orderDetails.OrderId)
	assert.Equal(t, expectedOrderDetails.UserId, orderDetails.UserId)
	assert.Equal(t, expectedOrderDetails.TotalPrice, orderDetails.TotalPrice)
	assert.Equal(t, expectedOrderDetails.Status, orderDetails.Status)

	sortOpt := cmpopts.SortSlices(func(a, b *order1.OrderItem) bool {
		return a.GetProductId() < b.GetProductId()
	})

	if !cmp.Equal(expectedOrderDetails.Items, orderDetails.Items, protocmp.Transform(), sortOpt) {
		diff := cmp.Diff(expectedOrderDetails.Items, orderDetails.Items, protocmp.Transform(), sortOpt)
		t.Errorf("OrderDetails.Items mismatch (-want +got):\n%s", diff)
	}
}

func TestClient_GetOrder_NotFound(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	orderID := int64(111)
	serverError := status.Error(codes.NotFound, "order not found")

	mockSrv.GetOrderFunc = func(ctx context.Context, req *order1.GetOrderRequest) (*order1.GetOrderResponse, error) {
		return nil, serverError
	}

	_, err := client.GetOrder(context.Background(), orderID)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, err.Error(), "grpc.order.get_order")
}

func TestClient_GetOrder_NilDetails(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	orderID := int64(123)
	mockSrv.GetOrderFunc = func(ctx context.Context, req *order1.GetOrderRequest) (*order1.GetOrderResponse, error) {
		return &order1.GetOrderResponse{OrderDetails: nil}, nil
	}

	orderDetails, err := client.GetOrder(context.Background(), orderID)

	assert.NoError(t, err)
	assert.Nil(t, orderDetails)
}

func TestClient_ListUserOrders_Success(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(33)
	expectedOrders := []*order1.OrderDetails{
		{OrderId: 10, UserId: userID, TotalPrice: 100, Items: nil, Status: ""},
		{OrderId: 11, UserId: userID, TotalPrice: 200, Items: nil, Status: ""},
	}

	mockSrv.ListUserOrdersFunc = func(ctx context.Context, req *order1.ListUserOrdersRequest) (*order1.ListUserOrdersResponse, error) {
		assert.Equal(t, userID, req.UserId)
		return &order1.ListUserOrdersResponse{Orders: expectedOrders}, nil
	}

	orders, err := client.ListUserOrders(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, orders)
	assert.Len(t, orders, len(expectedOrders))
	sortOpt := cmpopts.SortSlices(func(a, b *order1.OrderDetails) bool {
		return a.GetOrderId() < b.GetOrderId()
	})

	if !cmp.Equal(expectedOrders, orders, protocmp.Transform(), sortOpt) {
		diff := cmp.Diff(expectedOrders, orders, protocmp.Transform(), sortOpt)
		t.Errorf("ListUserOrders mismatch (-want +got):\n%s", diff)
	}
}

func TestClient_ListUserOrders_Empty(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(44)
	var expectedOrders []*order1.OrderDetails

	mockSrv.ListUserOrdersFunc = func(ctx context.Context, req *order1.ListUserOrdersRequest) (*order1.ListUserOrdersResponse, error) {
		return &order1.ListUserOrdersResponse{Orders: expectedOrders}, nil
	}

	orders, err := client.ListUserOrders(context.Background(), userID)

	assert.NoError(t, err)
	if expectedOrders == nil {
		assert.Nil(t, orders)
	} else {
		assert.NotNil(t, orders)
		assert.Empty(t, orders)
	}
}

func TestClient_ListUserOrders_ServerError(t *testing.T) {
	mockSrv := &mockOrderServer{}
	client, cleanup := setupTestOrderGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(55)
	serverError := status.Error(codes.Unavailable, "database connection lost")

	mockSrv.ListUserOrdersFunc = func(ctx context.Context, req *order1.ListUserOrdersRequest) (*order1.ListUserOrdersResponse, error) {
		return nil, serverError
	}

	_, err := client.ListUserOrders(context.Background(), userID)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Contains(t, err.Error(), "grpc.order.list_user_orders")
}
