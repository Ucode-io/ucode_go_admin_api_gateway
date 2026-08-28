FROM golang:1.23.2 as builder

RUN mkdir -p $GOPATH/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway 
WORKDIR $GOPATH/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway

COPY . ./

# installing depends and build
RUN export CGO_ENABLED=0 && \
    export GOOS=linux && \
    go mod vendor && \
    make build && \
    test -f .meta-ads.env.enc || touch .meta-ads.env.enc && \
    mv .meta-ads.env.enc / && \
    mv ./bin/ucode_go_admin_api_gateway /

FROM alpine
COPY --from=builder ucode_go_admin_api_gateway .
COPY --from=builder /meta-ads.env.enc /meta-ads.env.enc

ENTRYPOINT ["/ucode_go_admin_api_gateway"]
