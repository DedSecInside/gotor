FROM golang:1.24.7-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/gotor \
    ./cmd/main

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/gotor /gotor
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/gotor"]
CMD ["-s", "-server-host", "0.0.0.0", "-server-port", "8081", "-log-format", "json"]
