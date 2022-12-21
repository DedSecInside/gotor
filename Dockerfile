# Dockerfile.production

FROM registry.semaphoreci.com/golang:1.19 as builder

ENV APP_HOME /go/src/gotor

WORKDIR "$APP_HOME"

COPY src/ .
COPY go.mod .
COPY go.sum .
COPY vendor .
COPY .env .

RUN go mod download
RUN go mod verify
RUN go build -o gotor cmd/main/main.go

FROM registry.semaphoreci.com/golang:1.19

ENV APP_HOME /go/src/gotor
RUN mkdir -p "$APP_HOME"
WORKDIR "$APP_HOME"

COPY src/api/ api/
COPY src/pkg/ pkg/
COPY src/internal/ internal/
COPY src/cmd/ cmd/
COPY .env .env
COPY --from=builder "$APP_HOME"/gotor $APP_HOME

EXPOSE 8010
CMD ["./gotor", "-server"]