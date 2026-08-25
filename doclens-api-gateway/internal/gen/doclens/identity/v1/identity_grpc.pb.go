package identityv1

import (
	"context"

	"google.golang.org/grpc"
)

const IdentityService_ServiceDesc_FullMethodName = "/doclens.identity.v1.IdentityService/"

type IdentityServiceClient interface {
	CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*User, error)
	Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error)
}

type identityServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewIdentityServiceClient(cc grpc.ClientConnInterface) IdentityServiceClient {
	return &identityServiceClient{cc: cc}
}

func (c *identityServiceClient) CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*User, error) {
	out := new(User)
	err := c.cc.Invoke(ctx, IdentityService_ServiceDesc_FullMethodName+"CreateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityServiceClient) Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, IdentityService_ServiceDesc_FullMethodName+"Login", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
