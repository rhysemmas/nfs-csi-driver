# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /csi-driver ./cmd/csi-driver

# Runtime stage (minimal; node needs mount/umount so use full env or distroless with cap)
FROM alpine:3.19
RUN apk add --no-cache ca-certificates util-linux nfs-utils
COPY --from=builder /csi-driver /csi-driver
ENTRYPOINT ["/csi-driver"]
