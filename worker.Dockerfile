# 两阶段构建 worker
FROM golang:1.14-alpine3.11 as BuildImage
ENV WORKDIR /clock
WORKDIR $WORKDIR
COPY . .
RUN apk update && apk add --no-cache git ca-certificates make bash
RUN wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz \
    && tar -Jxf upx-3.96-amd64_linux.tar.xz \
    && cp upx-3.96-amd64_linux/upx /usr/local/bin/
RUN make build-worker && make upx-worker

FROM alpine:3.11
# 设置固定的项目路径
ENV WORKDIR /clock
WORKDIR $WORKDIR

RUN apk add --no-cache ca-certificates tzdata bash
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

# 添加应用可执行文件，并设置执行权限
RUN mkdir $WORKDIR/bin
# dst WORKDIR 后面的 / 需要加
COPY --from=BuildImage $WORKDIR/bin/* $WORKDIR/

# TODO: 分环境/加个 release
CMD ./worker -c $WORKDIR/dev.yaml
