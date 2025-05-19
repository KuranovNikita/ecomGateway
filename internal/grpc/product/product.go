package productgrpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	product1 "github.com/KuranovNikita/ecomProto/gen/go/product"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	api product1.ProductServiceClient
	log *slog.Logger
}

func New(
	log *slog.Logger,
	target string,
	timeout time.Duration,
	retriesCount int,
	additionalOpts ...grpc.DialOption,
) (*Client, error) {
	const op = "grpc.product.New"

	retryInterceptorOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	var dialOpts []grpc.DialOption

	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(
		grpcretry.UnaryClientInterceptor(retryInterceptorOpts...),
	))

	dialOpts = append(dialOpts, additionalOpts...)

	cc, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create gRPC client: %w", op, err)
	}

	return &Client{
		api: product1.NewProductServiceClient(cc),
	}, nil
}

func (c *Client) GetProduct(ctx context.Context, productID int64) (*product1.ProductDetails, error) {
	const op = "grpc.product.get_product"

	resp, err := c.api.GetProduct(ctx, &product1.GetProductRequest{
		ProductId: productID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp.ProductDetails, nil
}

func (c *Client) ListProducts(ctx context.Context, filter string) ([]*product1.ProductDetails, error) {
	const op = "grpc.product.list_products"

	resp, err := c.api.ListProducts(ctx, &product1.ListProductsRequest{
		Filter: filter,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp.Products, nil
}

func (c *Client) CheckStock(ctx context.Context, productID int64, quantity int32) (bool, error) {
	const op = "grpc.product.check_stock"

	resp, err := c.api.CheckStock(ctx, &product1.CheckStockRequest{
		ProductId: productID,
		Quantity:  quantity,
	})
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return resp.IsAvailable, nil
}

func (c *Client) UpdateStock(ctx context.Context, productID int64, quantityChange int32) error {
	const op = "grpc.product.update_stock"

	_, err := c.api.UpdateStock(ctx, &product1.UpdateStockRequest{
		ProductId:      productID,
		QuantityChange: quantityChange,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
