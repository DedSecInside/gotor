FROM golang:1.21-alpine

WORKDIR /go/src/gotor

# move over source code directories
COPY . .

RUN go mod download
RUN go mod verify
RUN go build -o gotor cmd/main/gotor.go

EXPOSE 8081
CMD ["./gotor", "-s"]
