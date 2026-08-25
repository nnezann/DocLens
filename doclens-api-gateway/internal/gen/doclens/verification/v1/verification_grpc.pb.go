package verificationv1

import (
	"context"

	"google.golang.org/grpc"
)

const VerificationService_ServiceDesc_FullMethodName = "/doclens.verification.v1.VerificationService/"

type VerificationServiceClient interface {
	StartVerification(ctx context.Context, in *StartVerificationRequest, opts ...grpc.CallOption) (*Verification, error)
	GetVerification(ctx context.Context, in *GetVerificationRequest, opts ...grpc.CallOption) (*Verification, error)
}

type verificationServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewVerificationServiceClient(cc grpc.ClientConnInterface) VerificationServiceClient {
	return &verificationServiceClient{cc: cc}
}

func (c *verificationServiceClient) StartVerification(ctx context.Context, in *StartVerificationRequest, opts ...grpc.CallOption) (*Verification, error) {
	out := new(Verification)
	err := c.cc.Invoke(ctx, VerificationService_ServiceDesc_FullMethodName+"StartVerification", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *verificationServiceClient) GetVerification(ctx context.Context, in *GetVerificationRequest, opts ...grpc.CallOption) (*Verification, error) {
	out := new(Verification)
	err := c.cc.Invoke(ctx, VerificationService_ServiceDesc_FullMethodName+"GetVerification", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
