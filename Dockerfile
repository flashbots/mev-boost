# syntax=docker/dockerfile:1
FROM golang:1.20 as builder
ARG VERSION
WORKDIR /build

COPY go.mod ./
COPY go.sum ./

RUN go mod download

ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -v \
    -ldflags "-w -s -X 'github.com/flashbots/mev-boost/config.Version=$VERSION'" \
    -o mev-boost .

FROM alpine
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/mev-boost /app/mev-boost
# Set runner as a non existent user
USER 65534:65534
EXPOSE 18550
ENTRYPOINT ["/app/mev-boost"]
