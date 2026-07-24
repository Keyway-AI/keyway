# syntax=docker/dockerfile:1
# Build context is the repo root (see bench/l2/docker-compose.yml).
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/l2 ./bench/l2

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/l2 /usr/local/bin/l2
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/l2"]
