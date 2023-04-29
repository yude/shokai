FROM golang:1.20.3-bullseye AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./
RUN go build -o /shokai

FROM debian:bullseye AS runner

WORKDIR /app

COPY --from=builder /app/shokai ./
COPY config.toml ./

EXPOSE 3000

CMD [ "/app/shokai" ]
