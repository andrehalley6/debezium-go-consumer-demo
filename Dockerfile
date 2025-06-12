FROM golang:1.23-alpine

WORKDIR /app

# install git and SSL root certs for go mod and HTTPS access
RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o go-consumer .

CMD ["./go-consumer"]
