FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server/

FROM gcr.io/distroless/static-debian12

COPY --from=builder /server /server

EXPOSE 9999

ENTRYPOINT ["/server"]
