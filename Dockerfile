FROM golang:1.25.1-trixie AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./
RUN go build -o shokai

FROM golang:1.25.1-trixie AS runner

WORKDIR /app

COPY --from=builder /app/shokai ./

EXPOSE 3000

CMD [ "/app/shokai" ]
