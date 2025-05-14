package usergrpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	user1 "github.com/KuranovNikita/ecomProto/gen/go/user"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	api user1.UserServiceClient
	log *slog.Logger
}

type UserDetails struct {
	UserID int64
	Login  string
	Email  string
}

func New(
	log *slog.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
) (*Client, error) {
	const op = "grpc.user.New"
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
		api: user1.NewUserServiceClient(cc),
	}, nil
}

func (c *Client) Register(ctx context.Context, email string, login string, password string) (int64, error) {
	const op = "grpc.user.register"
	resp, err := c.api.Register(ctx, &user1.RegisterRequest{
		Email:    email,
		Login:    login,
		Password: password,
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return resp.UserId, nil
}

func (c *Client) Login(ctx context.Context, login string, password string) (string, error) {
	const op = "grpc.user.login"
	resp, err := c.api.Login(ctx, &user1.LoginRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return resp.Token, nil
}

func (c *Client) GetUser(ctx context.Context, userID int64) (*user1.UserDetails, error) {
	const op = "grpc.user.getUser"
	resp, err := c.api.GetUser(ctx, &user1.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if resp.UserDetails == nil {
		return nil, fmt.Errorf("%s: user details are empty", op)
	}

	return resp.UserDetails, nil
}
