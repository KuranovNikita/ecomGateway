package ordergrpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	order1 "github.com/KuranovNikita/ecomProto/gen/go/order"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	api order1.OrderServiceClient
	log *slog.Logger
}

func New(
	log *slog.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
) (*Client, error) {
	const op = "grpc.order.New"
	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	cc, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpcretry.UnaryClientInterceptor(retryOpts...),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Client{
		api: order1.NewOrderServiceClient(cc),
	}, nil
}

func (c *Client) CreateOrder(ctx context.Context, userID int64, items []*order1.OrderItem) (int64, int64, error) {
	const op = "grpc.order.create_order"

	resp, err := c.api.CreateOrder(ctx, &order1.CreateOrderRequest{
		UserId: userID,
		Items:  items,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", op, err)
	}

	return resp.OrderId, resp.TotalPrice, nil
}

func (c *Client) GetOrder(ctx context.Context, orderID int64) (*order1.OrderDetails, error) {
	const op = "grpc.order.get_order"

	resp, err := c.api.GetOrder(ctx, &order1.GetOrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp.OrderDetails, nil

}

func (c *Client) ListUserOrders(ctx context.Context, userID int64) ([]*order1.OrderDetails, error) {
	const op = "grpc.order.list_user_orders"

	resp, err := c.api.ListUserOrders(ctx, &order1.ListUserOrdersRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp.Orders, nil
}

func NewOrderItem(productID int64, quantity int32, price int64) *order1.OrderItem {
	return &order1.OrderItem{
		ProductId: productID,
		Quantity:  quantity,
		Price:     price,
	}
}
