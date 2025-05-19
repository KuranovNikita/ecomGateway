package usergrpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	user1 "github.com/KuranovNikita/ecomProto/gen/go/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockUserServer struct {
	user1.UnimplementedUserServiceServer

	RegisterFunc func(ctx context.Context, req *user1.RegisterRequest) (*user1.RegisterResponse, error)
	LoginFunc    func(ctx context.Context, req *user1.LoginRequest) (*user1.LoginResponse, error)
	GetUserFunc  func(ctx context.Context, req *user1.GetUserRequest) (*user1.GetUserResponse, error)
}

func (s *mockUserServer) Register(ctx context.Context, req *user1.RegisterRequest) (*user1.RegisterResponse, error) {
	if s.RegisterFunc != nil {
		return s.RegisterFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}

func (s *mockUserServer) Login(ctx context.Context, req *user1.LoginRequest) (*user1.LoginResponse, error) {
	if s.LoginFunc != nil {
		return s.LoginFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}

func (s *mockUserServer) GetUser(ctx context.Context, req *user1.GetUserRequest) (*user1.GetUserResponse, error) {
	if s.GetUserFunc != nil {
		return s.GetUserFunc(ctx, req)
	}
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}

func setupTestGRPCServer(t *testing.T, mockSrv *mockUserServer) (*Client, func()) {
	t.Helper()

	bufSize := 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcServer := grpc.NewServer()
	user1.RegisterUserServiceServer(grpcServer, mockSrv)

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

func TestClient_Register_Success(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	expectedUserID := int64(123)
	email := "test@example.com"
	login := "testuser"
	password := "password"

	mockSrv.RegisterFunc = func(ctx context.Context, req *user1.RegisterRequest) (*user1.RegisterResponse, error) {
		assert.Equal(t, email, req.Email)
		assert.Equal(t, login, req.Login)
		assert.Equal(t, password, req.Password)
		return &user1.RegisterResponse{UserId: expectedUserID}, nil
	}

	userID, err := client.Register(context.Background(), email, login, password)

	assert.NoError(t, err)
	assert.Equal(t, expectedUserID, userID)
}

func TestClient_Register_ServerError(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	serverError := status.Error(codes.Internal, "database is down")

	mockSrv.RegisterFunc = func(ctx context.Context, req *user1.RegisterRequest) (*user1.RegisterResponse, error) {
		return nil, serverError
	}

	_, err := client.Register(context.Background(), "any", "any", "any")

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "database is down")
	assert.Contains(t, err.Error(), "grpc.user.register")
}

func TestClient_Login_Success(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	expectedToken := "test-jwt-token"
	login := "testuser"
	password := "password"

	mockSrv.LoginFunc = func(ctx context.Context, req *user1.LoginRequest) (*user1.LoginResponse, error) {
		assert.Equal(t, login, req.Login)
		assert.Equal(t, password, req.Password)
		return &user1.LoginResponse{Token: expectedToken}, nil
	}

	token, err := client.Login(context.Background(), login, password)

	assert.NoError(t, err)
	assert.Equal(t, expectedToken, token)
}

func TestClient_Login_InvalidCredentials(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	serverError := status.Error(codes.Unauthenticated, "invalid login or password")

	mockSrv.LoginFunc = func(ctx context.Context, req *user1.LoginRequest) (*user1.LoginResponse, error) {
		return nil, serverError
	}

	_, err := client.Login(context.Background(), "wrong", "wrong")

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, err.Error(), "grpc.user.login")
}

func TestClient_GetUser_Success(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(1)
	expectedDetails := &user1.UserDetails{
		UserId: userID,
		Login:  "testuser",
		Email:  "test@example.com",
	}

	mockSrv.GetUserFunc = func(ctx context.Context, req *user1.GetUserRequest) (*user1.GetUserResponse, error) {
		assert.Equal(t, userID, req.UserId)
		return &user1.GetUserResponse{UserDetails: expectedDetails}, nil
	}

	details, err := client.GetUser(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, details)
	assert.Equal(t, expectedDetails.UserId, details.UserId)
	assert.Equal(t, expectedDetails.Login, details.Login)
	assert.Equal(t, expectedDetails.Email, details.Email)
}

func TestClient_GetUser_NotFound(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(999)
	serverError := status.Error(codes.NotFound, "user not found")

	mockSrv.GetUserFunc = func(ctx context.Context, req *user1.GetUserRequest) (*user1.GetUserResponse, error) {
		return nil, serverError
	}

	_, err := client.GetUser(context.Background(), userID)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, err.Error(), "grpc.user.getUser")
}

func TestClient_GetUser_EmptyDetails(t *testing.T) {
	mockSrv := &mockUserServer{}
	client, cleanup := setupTestGRPCServer(t, mockSrv)
	defer cleanup()

	userID := int64(1)
	mockSrv.GetUserFunc = func(ctx context.Context, req *user1.GetUserRequest) (*user1.GetUserResponse, error) {
		return &user1.GetUserResponse{UserDetails: nil}, nil
	}

	_, err := client.GetUser(context.Background(), userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user details are empty")
	assert.Contains(t, err.Error(), "grpc.user.getUser")
}
