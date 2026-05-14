FROM golang:1.21-alpine AS builder
# ca-certificates nécessaires pour les appels HTTPS sortants (M3)
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o web-app .

FROM scratch
# Inclusion des certificats CA pour supporter les connexions TLS sortantes
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/web-app /web-app
EXPOSE 8080
ENTRYPOINT ["/web-app"]
