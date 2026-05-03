FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /javapi ./cmd/api
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /javapi /javapi
COPY configs/ /configs/
ENV GOMEMLIMIT=400MiB
EXPOSE 8080
ENTRYPOINT ["/javapi"]
