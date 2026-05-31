FROM golang:1.25-alpine AS build

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/player-stats-backend ./cmd/player-stats

FROM alpine:3.22

WORKDIR /app
COPY --from=build /out/player-stats-backend /app/player-stats-backend

ENV SERVER_PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/player-stats-backend"]
