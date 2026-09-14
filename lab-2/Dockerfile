FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY lib/go.mod lib/go.sum ./lib/
COPY lab-2/go.mod lab-2/go.sum ./lab-2/

RUN cd lab-2 && go mod download

COPY lib/ ./lib/
COPY lab-2/ ./lab-2/

RUN cd lab-2 && CGO_ENABLED=0 GOOS=linux go build -o /app/app ./cmd

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /app/app /app/app
COPY lab-2/numbers.txt /app/numbers.txt

USER app

CMD ["/app/app"]