# Dockerfile.production

FROM registry.semaphoreci.com/golang:1.19 as builder

ENV APP_HOME /go/src/gotor

WORKDIR "$APP_HOME"

# move over source code directories
COPY . .

RUN go mod download
RUN go mod verify
RUN go build -o gotor cmd/main/gotor.go

FROM registry.semaphoreci.com/golang:1.19

ENV APP_HOME /go/src/gotor
RUN mkdir -p "$APP_HOME"
WORKDIR "$APP_HOME"

COPY . .
COPY --from=builder "$APP_HOME"/gotor $APP_HOME

EXPOSE 8081
CMD ["./gotor", "-s"]
