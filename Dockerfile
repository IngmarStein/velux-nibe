FROM golang:1.16.2-alpine AS builder
WORKDIR /go/src/github.com/ingmarstein/velux-nibe/
COPY . .
RUN apk add -U --no-cache ca-certificates
RUN CGO_ENABLED=0 GOOS=linux go build .

FROM scratch
COPY --from=builder /go/src/github.com/ingmarstein/velux-nibe/velux-nibe /velux-nibe
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/velux-nibe"]
