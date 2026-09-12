FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o wormhole wormhole.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/wormhole /usr/local/bin/wormhole
ENTRYPOINT ["wormhole"]
