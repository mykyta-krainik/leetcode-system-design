FROM golang:1.23.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /problem-service main.go

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /problem-service /app/problem-service

EXPOSE 8080

CMD ["/app/problem-service"]
