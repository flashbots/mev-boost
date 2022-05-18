# syntax=docker/dockerfile:1
# FROM golang:1.18 as builder
FROM golang:1.18 as builder
WORKDIR /build
ADD . /build/
RUN --mount=type=cache,target=/root/.cache/go-build make build-for-docker

FROM golang:1.18
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/mev-boost /app/mev-boost
EXPOSE 18550
ENTRYPOINT ["/app/mev-boost"]
