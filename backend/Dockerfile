# Production image: static binary in a distroless runtime.
FROM golang:1.22-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api ./api
COPY --from=build /src/migrations ./migrations

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["./api"]
