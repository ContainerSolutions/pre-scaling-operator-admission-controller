# build stage
FROM golang:1.15-stretch AS build-env
RUN mkdir -p /go/src/github.com/containersol/prescale-operator-admission
WORKDIR /go/src/github.com/containersol/prescale-operator-admission
COPY  . .
RUN useradd -u 10001 webhook
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o webhook

FROM scratch
COPY --from=build-env /go/src/github.com/containersol/prescale-operator-admission/webhook .
COPY --from=build-env /etc/passwd /etc/passwd
USER webhook
ENTRYPOINT ["/webhook"]