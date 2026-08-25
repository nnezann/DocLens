package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/doclens/api-gateway/internal/observability"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func writeUpstreamError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, context.Canceled) {
		observability.Error(w, http.StatusRequestTimeout, "request canceled")
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		observability.Error(w, http.StatusGatewayTimeout, "upstream timeout")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		observability.Error(w, http.StatusBadGateway, "upstream error")
		return
	}
	switch st.Code() {
	case codes.InvalidArgument:
		observability.Error(w, http.StatusBadRequest, st.Message())
	case codes.Unauthenticated:
		observability.Error(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		observability.Error(w, http.StatusForbidden, st.Message())
	case codes.NotFound:
		observability.Error(w, http.StatusNotFound, st.Message())
	case codes.AlreadyExists, codes.FailedPrecondition:
		observability.Error(w, http.StatusConflict, st.Message())
	case codes.Unavailable:
		observability.Error(w, http.StatusServiceUnavailable, st.Message())
	case codes.DeadlineExceeded:
		observability.Error(w, http.StatusGatewayTimeout, st.Message())
	default:
		observability.Error(w, http.StatusBadGateway, st.Message())
	}
}
