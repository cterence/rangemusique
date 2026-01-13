FROM golang:1.25-bookworm@sha256:019c22232e57fda8ded2b10a8f201989e839f3d3f962d4931375069bbb927e03 AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.25-bookworm@sha256:019c22232e57fda8ded2b10a8f201989e839f3d3f962d4931375069bbb927e03 AS builder

WORKDIR /app
COPY --from=deps /go/pkg /go/pkg
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o main .

FROM golang:1.25-bookworm@sha256:019c22232e57fda8ded2b10a8f201989e839f3d3f962d4931375069bbb927e03 AS development

WORKDIR /app
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

FROM debian:bookworm-slim@sha256:94c4d598b5987d76c38408657aae7118b101662595bf5eefe478e093a0bed2f6 AS production

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  groupadd -r appuser && \
  useradd -r -g appuser appuser

COPY --from=builder /app/main .
RUN chown appuser:appuser /app/main

USER appuser

ENTRYPOINT ["/app/main"]
