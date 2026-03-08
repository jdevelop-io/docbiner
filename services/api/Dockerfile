# ---- Builder ----
FROM golang:1.25-alpine AS builder
WORKDIR /build

# Copy go.work and all module go.mod/go.sum files for dependency caching
COPY go.work ./
COPY cmd/cli/go.mod cmd/cli/go.sum* ./cmd/cli/
COPY internal/apikey/go.mod internal/apikey/go.sum* ./internal/apikey/
COPY internal/auth/go.mod internal/auth/go.sum* ./internal/auth/
COPY internal/billing/go.mod internal/billing/go.sum* ./internal/billing/
COPY internal/config/go.mod internal/config/go.sum* ./internal/config/
COPY internal/database/go.mod internal/database/go.sum* ./internal/database/
COPY internal/delivery/go.mod internal/delivery/go.sum* ./internal/delivery/
COPY internal/domain/go.mod internal/domain/go.sum* ./internal/domain/
COPY internal/pdfutil/go.mod internal/pdfutil/go.sum* ./internal/pdfutil/
COPY internal/queue/go.mod internal/queue/go.sum* ./internal/queue/
COPY internal/renderer/go.mod internal/renderer/go.sum* ./internal/renderer/
COPY internal/storage/go.mod internal/storage/go.sum* ./internal/storage/
COPY internal/tmpl/go.mod internal/tmpl/go.sum* ./internal/tmpl/
COPY internal/usage/go.mod internal/usage/go.sum* ./internal/usage/
COPY services/api/go.mod services/api/go.sum ./services/api/
COPY services/worker/go.mod services/worker/go.sum ./services/worker/

RUN cd services/api && go mod download

# Copy all source (go.work requires all workspace modules)
COPY cmd/cli/ ./cmd/cli/
COPY internal/ ./internal/
COPY services/api/ ./services/api/
COPY services/worker/ ./services/worker/

RUN cd services/api && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/api .

# ---- Runtime ----
FROM alpine:3.19
RUN apk add --no-cache ca-certificates chromium && \
    addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup
ENV CHROMIUM_PATH=/usr/bin/chromium-browser
COPY --from=builder /build/api /usr/local/bin/api
USER appuser
EXPOSE 8080
CMD ["api"]
