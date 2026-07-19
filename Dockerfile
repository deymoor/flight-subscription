FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/consumer ./cmd/consumer

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/consumer /app/consumer
COPY internal/storage/postgres/migrations /app/internal/storage/postgres/migrations
USER app
EXPOSE 8080
CMD ["/app/api"]
