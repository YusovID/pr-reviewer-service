FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPROXY=https://goproxy.io,direct

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

WORKDIR /app/cmd/migrator

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /migrator .

FROM alpine:3.22

WORKDIR /

COPY --from=builder /migrator /migrator
COPY ./migrations /migrations
COPY ./config/docker.yml /config/docker.yml

EXPOSE 8081

CMD ["/migrator"]