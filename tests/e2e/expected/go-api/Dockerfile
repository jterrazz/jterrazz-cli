FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/app ./cmd/...

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/app /bin/app
USER nobody
ENTRYPOINT ["/bin/app"]

