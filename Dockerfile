FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates libc6-compat

COPY --from=builder /app/main .

COPY .env /app/.env

COPY templates/ /app/templates/

EXPOSE 8080

CMD ["./main"]
