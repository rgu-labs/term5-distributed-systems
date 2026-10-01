FROM golang:1.27-alpine AS builder

ARG APP=pinger

WORKDIR /app

COPY lib/go.mod lib/go.sum ./lib/
COPY lab-3/go.mod lab-3/go.sum ./lab-3/

RUN cd lab-3 && go mod download

COPY lib/ ./lib/
COPY lab-3/ ./lab-3/

RUN cd lab-3 && CGO_ENABLED=0 GOOS=linux go build -o /app/app ./cmd/${APP}

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /app/app /app/app

USER app

CMD ["/app/app"]
