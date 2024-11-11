FROM golang:1.23.2

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o problem_management

EXPOSE 8080

CMD ["./problem_management"]
