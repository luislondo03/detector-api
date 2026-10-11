FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/driver-vision-api ./cmd/api

FROM alpine:3.21
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/driver-vision-api ./driver-vision-api
RUN mkdir -p /app/storage/evidence && chown -R app:app /app
USER app
ENV PORT=8080 EVIDENCE_DIRECTORY=/app/storage/evidence
EXPOSE 8080
CMD ["./driver-vision-api"]
