FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/document-intake ./cmd/document-intake

FROM alpine:3.21
RUN adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=build /out/document-intake /usr/local/bin/document-intake
COPY --from=build /src/migrations ./migrations
USER app
EXPOSE 9002 9092
ENTRYPOINT ["/usr/local/bin/document-intake"]
