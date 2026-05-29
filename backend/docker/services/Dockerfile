FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./services/${SERVICE_NAME}/cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /service /service
COPY --from=builder /app/config /config

EXPOSE 8080 50051

ENTRYPOINT ["/service"]
