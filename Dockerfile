FROM golang:1.10-alpine AS build
RUN apk add --update ca-certificates bash curl git
RUN curl https://raw.githubusercontent.com/golang/dep/v0.5.0/install.sh | sh

COPY . /go/src/github.com/danisla/appdb-operator/
WORKDIR /go/src/github.com/danisla/appdb-operator
RUN dep ensure
WORKDIR /go/src/github.com/danisla/appdb-operator/cmd/appdb-instance-operator
RUN go install
WORKDIR /go/src/github.com/danisla/appdb-operator/cmd/appdb-operator
RUN go install

FROM alpine:3.7
RUN apk add --update ca-certificates bash curl
RUN curl -sfSL https://storage.googleapis.com/kubernetes-release/release/v1.11.0/bin/linux/amd64/kubectl > /usr/bin/kubectl && chmod +x /usr/bin/kubectl
COPY --from=build /go/bin/appdb-instance-operator /usr/bin/
COPY --from=build /go/bin/appdb-operator /usr/bin/
COPY config/ /config/