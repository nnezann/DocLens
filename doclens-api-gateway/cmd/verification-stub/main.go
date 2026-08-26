package main

import (
	"context"
	"net"
	"os"

	verificationv1 "github.com/doclens/api-gateway/internal/gen/doclens/verification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type server struct {
}

type verificationServer interface {
	StartVerification(context.Context, *verificationv1.StartVerificationRequest) (*verificationv1.Verification, error)
	GetVerification(context.Context, *verificationv1.GetVerificationRequest) (*verificationv1.Verification, error)
}

func (server) StartVerification(context.Context, *verificationv1.StartVerificationRequest) (*verificationv1.Verification, error) {
	return nil, status.Error(codes.Unimplemented, "verification service is not implemented in this local stack")
}

func (server) GetVerification(context.Context, *verificationv1.GetVerificationRequest) (*verificationv1.Verification, error) {
	return nil, status.Error(codes.Unimplemented, "verification service is not implemented in this local stack")
}

func main() {
	address := os.Getenv("VERIFICATION_GRPC_ADDR")
	if address == "" {
		address = ":9003"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	stub := server{}
	grpcServer.RegisterService(&grpc.ServiceDesc{
		ServiceName: "doclens.verification.v1.VerificationService",
		HandlerType: (*verificationServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "StartVerification", Handler: func(srv any, ctx context.Context, decoder func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
				req := new(verificationv1.StartVerificationRequest)
				if err := decoder(req); err != nil {
					return nil, err
				}
				return srv.(verificationServer).StartVerification(ctx, req)
			}},
			{MethodName: "GetVerification", Handler: func(srv any, ctx context.Context, decoder func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
				req := new(verificationv1.GetVerificationRequest)
				if err := decoder(req); err != nil {
					return nil, err
				}
				return srv.(verificationServer).GetVerification(ctx, req)
			}},
		},
	}, stub)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)
	if err := grpcServer.Serve(listener); err != nil {
		panic(err)
	}
}
