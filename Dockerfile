FROM node:22-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/dist ./ui/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o g0router ./cmd/g0router

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /app/g0router /g0router
VOLUME ["/data"]
EXPOSE 20128
ENV DATA_DIR=/data
ENV PORT=20128
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/g0router", "healthcheck"]
ENTRYPOINT ["/g0router"]
CMD ["serve"]
