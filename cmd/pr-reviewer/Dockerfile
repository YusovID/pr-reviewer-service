FROM golang:1.25.4-alpine3.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPROXY=https://goproxy.io,direct

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

WORKDIR /app/cmd/pr-reviewer

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /pr-reviewer . 

FROM alpine:3.22

WORKDIR /

COPY --from=builder /pr-reviewer /pr-reviewer
COPY ./config/docker.yml /config/docker.yml

EXPOSE 8080

CMD ["/pr-reviewer"]