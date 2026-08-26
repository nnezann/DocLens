FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/identity ./cmd/identity

FROM alpine:3.21
RUN adduser -D -H -u 10001 app
COPY --from=build /out/identity /usr/local/bin/identity
USER app
EXPOSE 9001
ENTRYPOINT ["/usr/local/bin/identity"]
