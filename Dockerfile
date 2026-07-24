# syntax=docker/dockerfile:1

# ---- Web build stage -------------------------------------------------------
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

# ---- Go build stage --------------------------------------------------------
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# Embed the built web assets so the single binary can serve the UI.
COPY --from=web /web/dist ./internal/api/webdist
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
      -X github.com/nometria/keyway/internal/version.Version=${VERSION} \
      -X github.com/nometria/keyway/internal/version.Commit=${COMMIT} \
      -X github.com/nometria/keyway/internal/version.Date=${DATE}" \
    -o /out/keyway ./cmd/keyway
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w" -o /out/keyway-runner ./cmd/keyway-runner

# ---- Runtime stage (distroless) --------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/keyway /usr/local/bin/keyway
COPY --from=build /out/keyway-runner /usr/local/bin/keyway-runner
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/keyway-runner"]
CMD ["serve"]
