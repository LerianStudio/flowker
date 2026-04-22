# syntax=docker/dockerfile:1.4

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

# Install required packages: git is needed for private modules
RUN apk add --no-cache git

WORKDIR /flowker

# Copy only go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./

# Download Go modules using GitHub token from BuildKit secret (needed for private modules)
RUN --mount=type=secret,id=github_token \
  GITHUB_TOKEN=$(cat /run/secrets/github_token 2>/dev/null || true) && \
  if [ -n "$GITHUB_TOKEN" ]; then \
    git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; \
  fi && \
  GOPRIVATE=github.com/LerianStudio/* go mod download

COPY . .

ARG TARGETPLATFORM
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d'/' -f2) go build -a -tags netgo -ldflags '-w -s -extldflags "-static"' -o /app cmd/app/main.go

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app /app
COPY --from=builder /flowker/migrations /migrations

# Run as non-root user for security hardening
USER nonroot:nonroot

EXPOSE 3005 7001

ENTRYPOINT ["/app"]



