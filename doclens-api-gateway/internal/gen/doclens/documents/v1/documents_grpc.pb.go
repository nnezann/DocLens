package documentsv1

import (
	"context"

	"google.golang.org/grpc"
)

const DocumentIntakeService_ServiceDesc_FullMethodName = "/doclens.documents.v1.DocumentIntakeService/"

type DocumentIntakeServiceClient interface {
	CreateDocument(ctx context.Context, in *CreateDocumentRequest, opts ...grpc.CallOption) (*Document, error)
	GetDocument(ctx context.Context, in *GetDocumentRequest, opts ...grpc.CallOption) (*Document, error)
}

type documentIntakeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDocumentIntakeServiceClient(cc grpc.ClientConnInterface) DocumentIntakeServiceClient {
	return &documentIntakeServiceClient{cc: cc}
}

func (c *documentIntakeServiceClient) CreateDocument(ctx context.Context, in *CreateDocumentRequest, opts ...grpc.CallOption) (*Document, error) {
	out := new(Document)
	err := c.cc.Invoke(ctx, DocumentIntakeService_ServiceDesc_FullMethodName+"CreateDocument", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *documentIntakeServiceClient) GetDocument(ctx context.Context, in *GetDocumentRequest, opts ...grpc.CallOption) (*Document, error) {
	out := new(Document)
	err := c.cc.Invoke(ctx, DocumentIntakeService_ServiceDesc_FullMethodName+"GetDocument", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
