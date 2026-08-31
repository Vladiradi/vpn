# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/panel ./cmd/panel

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/panel /panel

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/panel"]
