# -------- Build runtime --------
FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY runtime.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o runtime runtime.go


# -------- Final runtime image --------
FROM alpine:latest

WORKDIR /runtime

COPY --from=builder /build/runtime .

# Required for your runtime
RUN mkdir -p /function

EXPOSE 8080

HEALTHCHECK --interval=2s --timeout=2s --retries=5 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./runtime"]