FROM golang:1.23.3

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cmd/ ./cmd

COPY pkg/ ./pkg

RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin/information-gathering ./cmd/main.go

EXPOSE 8001

CMD ["./bin/information-gathering"]
