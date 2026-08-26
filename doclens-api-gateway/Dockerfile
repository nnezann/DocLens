FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/verification-stub ./cmd/verification-stub

FROM alpine:3.21 AS gateway
RUN adduser -D -H -u 10001 app
COPY --from=build /out/gateway /usr/local/bin/gateway
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]

FROM alpine:3.21 AS verification-stub
RUN adduser -D -H -u 10001 app
COPY --from=build /out/verification-stub /usr/local/bin/verification-stub
USER app
EXPOSE 9003
ENTRYPOINT ["/usr/local/bin/verification-stub"]
