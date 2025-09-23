FROM golang:1.25 AS builder
WORKDIR /src

# Cache dependencies early
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags "-s -w" -o /out/mocker ./cmd/mocker

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=builder /out/mocker /usr/local/bin/mocker

ENTRYPOINT ["/usr/local/bin/mocker"]
CMD ["serve", "-c", "/app/config.yaml", "-p"]
