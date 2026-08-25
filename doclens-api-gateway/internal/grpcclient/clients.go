package grpcclient

import (
	"context"
	"errors"
	"time"

	documentsv1 "github.com/doclens/api-gateway/internal/gen/doclens/documents/v1"
	identityv1 "github.com/doclens/api-gateway/internal/gen/doclens/identity/v1"
	verificationv1 "github.com/doclens/api-gateway/internal/gen/doclens/verification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

type Clients struct {
	Identity     identityv1.IdentityServiceClient
	Documents    documentsv1.DocumentIntakeServiceClient
	Verification verificationv1.VerificationServiceClient
	HealthChecks map[string]healthv1.HealthClient
	conns        []*grpc.ClientConn
}

type Targets struct {
	Identity     string
	Documents    string
	Verification string
	Insecure     bool
}

func Dial(ctx context.Context, targets Targets) (*Clients, error) {
	if !targets.Insecure {
		return nil, errors.New("TLS gRPC dialing is not configured yet; set GRPC_INSECURE=true for local development")
	}
	dial := func(target string) (*grpc.ClientConn, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	}
	identityConn, err := dial(targets.Identity)
	if err != nil {
		return nil, err
	}
	documentsConn, err := dial(targets.Documents)
	if err != nil {
		identityConn.Close()
		return nil, err
	}
	verificationConn, err := dial(targets.Verification)
	if err != nil {
		identityConn.Close()
		documentsConn.Close()
		return nil, err
	}
	return &Clients{
		Identity:     identityv1.NewIdentityServiceClient(identityConn),
		Documents:    documentsv1.NewDocumentIntakeServiceClient(documentsConn),
		Verification: verificationv1.NewVerificationServiceClient(verificationConn),
		HealthChecks: map[string]healthv1.HealthClient{
			"identity":     healthv1.NewHealthClient(identityConn),
			"documents":    healthv1.NewHealthClient(documentsConn),
			"verification": healthv1.NewHealthClient(verificationConn),
		},
		conns: []*grpc.ClientConn{identityConn, documentsConn, verificationConn},
	}, nil
}

func (c *Clients) Close() error {
	var joined error
	for _, conn := range c.conns {
		joined = errors.Join(joined, conn.Close())
	}
	return joined
}
