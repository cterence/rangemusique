FROM golang:1.26-bookworm@sha256:eae3cdfa040d0786510a5959d36a836978724d03b34a166ba2e0e198baac9196 AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.26-bookworm@sha256:eae3cdfa040d0786510a5959d36a836978724d03b34a166ba2e0e198baac9196 AS builder

WORKDIR /app
COPY --from=deps /go/pkg /go/pkg
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o main .

FROM golang:1.26-bookworm@sha256:eae3cdfa040d0786510a5959d36a836978724d03b34a166ba2e0e198baac9196 AS development

WORKDIR /app
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

FROM debian:bookworm-slim@sha256:74d56e3931e0d5a1dd51f8c8a2466d21de84a271cd3b5a733b803aa91abf4421 AS production

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  groupadd -r appuser && \
  useradd -r -g appuser appuser

COPY --from=builder /app/main .
RUN chown appuser:appuser /app/main

USER appuser

ENTRYPOINT ["/app/main"]
