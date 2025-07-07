FROM golang:1.21.0 as builder

RUN mkdir -p $GOPATH/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway 
WORKDIR $GOPATH/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway

COPY . ./

# installing depends and build
RUN export CGO_ENABLED=1 && \
    export GOOS=linux && \
    go mod vendor && \
    make build && \
    mv ./bin/ucode_go_admin_api_gateway /

ENTRYPOINT ["/ucode_go_admin_api_gateway"]
