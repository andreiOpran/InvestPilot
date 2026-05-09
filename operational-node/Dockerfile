# compile in a Go image
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# ccopy binary into a minimal runtime image
FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/internal/mailer/assets ./internal/mailer/assets
EXPOSE 8081
CMD ["./server"]