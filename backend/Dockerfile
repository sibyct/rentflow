# Production image: static binary in a distroless runtime.
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api
# Separate one-shot binary invoked by fly.toml/fly.staging.toml's
# [deploy] release_command — see cmd/migrate/main.go's doc comment for
# why this exists as its own binary rather than running on API startup.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# Background worker (scheduled rent / late fees / recurring expenses /
# owner statements, and the email outbox). Same image as the API, run as
# its own Fly process group — see fly.toml's [processes].
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api ./api
COPY --from=build /out/migrate ./migrate
COPY --from=build /out/worker ./worker
COPY --from=build /src/migrations ./migrations

USER nonroot:nonroot
EXPOSE 8080
# CMD, not ENTRYPOINT: Fly's [processes] and release_command each replace
# the command, and with an ENTRYPOINT set they would be appended to it as
# arguments (e.g. ./api /app/worker) instead of running.
CMD ["./api"]
