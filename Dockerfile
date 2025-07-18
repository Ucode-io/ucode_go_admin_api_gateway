FROM golang:1.23.2 as builder

RUN mkdir -p $GOPATH/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway 
WORKDIR $GOPATH/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway

COPY . ./

# installing depends and build
RUN export CGO_ENABLED=0 && \
    export GOOS=linux && \
    go mod vendor && \
    make build && \
    mv ./bin/ucode_go_admin_api_gateway /

RUN apt-get update && apt-get install -y \
    wkhtmltopdf \
 && apt-get clean \
 && bash \
 && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/ucode_go_admin_api_gateway"]
