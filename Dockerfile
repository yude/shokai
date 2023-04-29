FROM golang:alpine3.17 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./
RUN go build -o shokai

FROM golang:alpine3.17 AS runner

WORKDIR /app

COPY --from=builder /app/shokai ./
COPY config.toml ./

EXPOSE 3000

CMD [ "/app/shokai" ]
