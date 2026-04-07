FROM golang:1.26-bookworm@sha256:8e8aa801e8417ef0b5c42b504dd34db3db911bb73dba933bd8bde75ed815fdbb AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.26-bookworm@sha256:8e8aa801e8417ef0b5c42b504dd34db3db911bb73dba933bd8bde75ed815fdbb AS builder

WORKDIR /app
COPY --from=deps /go/pkg /go/pkg
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o main .

FROM golang:1.26-bookworm@sha256:8e8aa801e8417ef0b5c42b504dd34db3db911bb73dba933bd8bde75ed815fdbb AS development

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
