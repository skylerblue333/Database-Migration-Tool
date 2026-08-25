FROM golang:1.25.1-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/sky-migrations ./main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/sky-migrations /sky-migrations
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/sky-migrations"]
