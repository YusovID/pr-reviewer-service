FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod tidy

COPY . .

WORKDIR /app/cmd/migrator

RUN CGO_ENABLED=0 GOOS=linux go build -o /migrator .

FROM alpine:3.22

WORKDIR /

COPY --from=builder /migrator /migrator
COPY ./migrations /migrations
COPY ./config/docker.yml /config/docker.yml

EXPOSE 8081

CMD ["/migrator"]