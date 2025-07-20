FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o email-catch ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/email-catch .
COPY --from=builder /app/config ./config

RUN mkdir -p /root/emails /root/logs

EXPOSE 25 587 465 2525

CMD ["./email-catch", "-config", "config/config.yaml"]