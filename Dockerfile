FROM golang:1.26-bookworm@sha256:09f72a3e4d00f209358f03b93e4d62e6ed45b786569c2d97e83cb7cbaaed15f2 AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.26-bookworm@sha256:09f72a3e4d00f209358f03b93e4d62e6ed45b786569c2d97e83cb7cbaaed15f2 AS builder

WORKDIR /app
COPY --from=deps /go/pkg /go/pkg
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o main .

FROM golang:1.26-bookworm@sha256:09f72a3e4d00f209358f03b93e4d62e6ed45b786569c2d97e83cb7cbaaed15f2 AS development

WORKDIR /app
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

FROM debian:bookworm-slim@sha256:4724b8cc51e33e398f0e2e15e18d5ec2851ff0c2280647e1310bc1642182655d AS production

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  groupadd -r appuser && \
  useradd -r -g appuser appuser

COPY --from=builder /app/main .
RUN chown appuser:appuser /app/main

USER appuser

ENTRYPOINT ["/app/main"]
