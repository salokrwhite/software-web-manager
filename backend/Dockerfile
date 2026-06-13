FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.20
WORKDIR /app
RUN adduser -D app
USER app
COPY --from=builder /bin/api /app/api
COPY backend/migrations /app/migrations
COPY backend/third_party/ip2region/data /app/third_party/ip2region/data
EXPOSE 8080
ENV HTTP_ADDR=:8080
CMD ["/app/api"]
