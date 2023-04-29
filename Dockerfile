FROM golang:alpine3.17 AS build
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY main.go go.mod ./
RUN go build .

FROM alpine:latest
WORKDIR /opt/app
COPy --from=build /go/src/app/pipeline ./
CMD [ "./pipeline" ]
