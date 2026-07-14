FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./services/${SERVICE_NAME}/cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# ARG scope resets per stage — re-declare to reference the service migrations.
ARG SERVICE_NAME
COPY --from=builder /service /service
COPY --from=builder /app/config /config
# Ship the service's own migrations so auto_migrate can apply them on start.
COPY --from=builder /app/services/${SERVICE_NAME}/migrations /migrations

EXPOSE 8080 50051

ENTRYPOINT ["/service"]
