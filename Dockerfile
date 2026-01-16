FROM golang:1.25-bookworm@sha256:2f768d462dbffbb0f0b3a5171009f162945b086f326e0b2a8fd5d29c3219ff14 AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.25-bookworm@sha256:2f768d462dbffbb0f0b3a5171009f162945b086f326e0b2a8fd5d29c3219ff14 AS builder

WORKDIR /app
COPY --from=deps /go/pkg /go/pkg
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o main .

FROM golang:1.25-bookworm@sha256:2f768d462dbffbb0f0b3a5171009f162945b086f326e0b2a8fd5d29c3219ff14 AS development

WORKDIR /app
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

FROM debian:bookworm-slim@sha256:56ff6d36d4eb3db13a741b342ec466f121480b5edded42e4b7ee850ce7a418ee AS production

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  groupadd -r appuser && \
  useradd -r -g appuser appuser

COPY --from=builder /app/main .
RUN chown appuser:appuser /app/main

USER appuser

ENTRYPOINT ["/app/main"]
