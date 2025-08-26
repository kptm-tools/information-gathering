# Stage 1: Builder - for compiling dependencies
FROM golang:1.24.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cmd/ ./cmd
COPY pkg/ ./pkg

# Install the 'air' live-reloading tool
RUN go install github.com/air-verse/air@latest

# Stage 2: Development - for running the application with live reload
FROM golang:1.24.4 AS development

WORKDIR /app

COPY --from=builder /go/pkg/mod /go/pkg/mod
COPY --from=builder /go/bin/air /go/bin/air

COPY .air.toml .
COPY Makefile .

COPY . .

EXPOSE 8001

# The command to run the application using air for live-reloading
CMD ["air"]

# Stage 3: Production builder - for building the production binary
FROM golang:1.24.4 AS production-builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cmd/ ./cmd
COPY pkg/ ./pkg

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ./bin/information-gathering ./cmd/main.go

# Stage 4: Production - minimal final image
FROM alpine:3.19 AS production

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=production-builder /app/bin/information-gathering /app/information-gathering

# Copy the wordlists from the production-builder stage
COPY --from=production-builder /app/pkg/services/wordlists /app/pkg/services/wordlists

EXPOSE 8001

USER nobody

CMD ["/app/information-gathering"]
