# 构建脚本

export GO111MODULE=on
export GOPROXY=https://goproxy.io

# set-env copy-config 在这里被依赖 在 build-master 和 build-worker 也被依赖，但是不会执行两次
.PHONY: build
build: set-env copy-config build-master upx-master build-worker upx-worker

# mac date doesn't have --rfc-3339=seconds
.PHONY: build-master
build-master: set-env copy-config
	go build -v -ldflags "-X 'main.goVersion=$$(go version)' \
	-X 'main.gitHash=$$(git show -s --format=%H)' \
	-X 'main.buildTime=$$(date)'" \
	-o bin/master master/main.go
	@echo "build master success"

.PHONY: build-worker
build-worker: set-env copy-config
	go build -v -ldflags "-X 'main.goVersion=$$(go version)' \
	-X 'main.gitHash=$$(git show -s --format=%H)' \
	-X 'main.buildTime=$$(date)'" \
	-o bin/worker worker/main.go
	@echo "build worker success"

.PHONY: copy-config
copy-config:
	rm -rf bin && mkdir -p bin && cp config/*.yaml bin/
	@echo "copy config success"

.PHONY: set-env
set-env:
	@echo "set env success"

.PHONY: docker-build-master
docker-build-master:
	docker build -f master.Dockerfile -t cron-master:${version} .

.PHONY: docker-build-worker
docker-build-worker:
	docker build -f worker.Dockerfile -t cron-worker:${version} .

# 删除无用的 none 镜像 先删除可能跑的容器，后删除镜像
# 想使用真正的 $ 需要用 $$
# TODO: 删除不存在的镜像/被容器占用的镜像 会有问题
.PHONY: remove-none-images
remove-none-images:
	docker images | awk '$$1=="<none>"' | awk '{print $$3}' | xargs docker rmi

# NOTICE: 需要确保有安装 upx
.PHONY: upx-master
upx-master:
	upx -v bin/master

# NOTICE: 需要确保有安装 upx
.PHONY: upx-worker
upx-worker:
	upx -v bin/worker
