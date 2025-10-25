FROM golang:1.25-bookworm@sha256:51b6b12427dc03451c24f7fc996c43a20e8a8e56f0849dd0db6ff6e9225cc892 AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.25-bookworm@sha256:51b6b12427dc03451c24f7fc996c43a20e8a8e56f0849dd0db6ff6e9225cc892 AS builder

WORKDIR /app
COPY --from=deps /go/pkg /go/pkg
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o main .

FROM golang:1.25-bookworm@sha256:51b6b12427dc03451c24f7fc996c43a20e8a8e56f0849dd0db6ff6e9225cc892 AS development

WORKDIR /app
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

FROM debian:bookworm-slim@sha256:78d2f66e0fec9e5a39fb2c72ea5e052b548df75602b5215ed01a17171529f706 AS production

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  groupadd -r appuser && \
  useradd -r -g appuser appuser

COPY --from=builder /app/main .
RUN chown appuser:appuser /app/main

USER appuser

ENTRYPOINT ["/app/main"]
