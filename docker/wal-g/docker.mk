# To use buildx: https://github.com/docker/buildx#docker-ce
export DOCKER_CLI_EXPERIMENTAL=enabled

# Docker image build and push setting
DOCKER:=DOCKER_BUILDKIT=1 docker
DOCKERFILE_DIR?=./docker
BUILDX_ENABLED ?= true
BUILDX_PLATFORMS ?= linux/amd64,linux/arm64
VERSION ?= "dev"
TAG_LATEST ?= false

# Image URL to use all building/pushing image targets
MYSQL_IMG ?= docker.io/apecloud/wal-g-mysql
PG_IMG ?= docker.io/apecloud/wal-g-pg
MONGO_IMG ?= docker.io/apecloud/wal-g-mongo

DOCKERFILE_DIR = ./docker/wal-g
DOCKER_BUILD_ARGS := --build-arg BUILD_DATE=$(shell date -u +%Y.%m.%d_%H:%M:%S) \
  --build-arg GIT_COMMIT_ID=$(shell git rev-parse --short HEAD) \
  --build-arg GIT_TAG_VERSION=$(shell git tag -l --points-at HEAD) \
  --build-arg BUILD_TAGS="$(BUILD_TAGS)"

build_mysql_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
	$(DOCKER) build . $(DOCKER_BUILD_ARGS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql --tag $(MYSQL_IMG):$(VERSION) --tag $(MYSQL_IMG):latest
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql --tag $(MYSQL_IMG):$(VERSION) --tag $(MYSQL_IMG):latest
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql --tag $(MYSQL_IMG):$(VERSION)
endif
endif

push_mysql_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
ifeq ($(TAG_LATEST), true)
	$(DOCKER) push $(MYSQL_IMG):latest
else
	$(DOCKER) push $(MYSQL_IMG):$(VERSION)
endif
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql --tag $(MYSQL_IMG):latest --push
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql --tag $(MYSQL_IMG):$(VERSION) --push
endif
endif

build_mysql_ubuntu_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
	$(DOCKER) build . $(DOCKER_BUILD_ARGS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql-ubuntu --tag $(MYSQL_IMG):$(VERSION)-ubuntu --tag $(MYSQL_IMG):latest-ubuntu
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql-ubuntu --tag $(MYSQL_IMG):latest-ubuntu
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql-ubuntu --tag $(MYSQL_IMG):$(VERSION)-ubuntu
endif
endif

push_mysql_ubuntu_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
ifeq ($(TAG_LATEST), true)
	$(DOCKER) push $(MYSQL_IMG):latest-ubuntu
else
	$(DOCKER) push $(MYSQL_IMG):$(VERSION)-ubuntu
endif
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql-ubuntu --tag $(MYSQL_IMG):latest-ubuntu --push
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mysql-ubuntu --tag $(MYSQL_IMG):$(VERSION)-ubuntu --push
endif
endif

build_pg_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
	$(DOCKER) build . $(DOCKER_BUILD_ARGS) --file $(DOCKERFILE_DIR)/Dockerfile-pg --tag $(PG_IMG):$(VERSION) --tag $(PG_IMG):latest
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-pg --tag $(PG_IMG):latest
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-pg --tag $(PG_IMG):$(VERSION)
endif
endif

push_pg_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
ifeq ($(TAG_LATEST), true)
	$(DOCKER) push $(PG_IMG):latest
else
	$(DOCKER) push $(PG_IMG):$(VERSION)
endif
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-pg --tag $(PG_IMG):latest --push
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-pg --tag $(PG_IMG):$(VERSION) --push
endif
endif

build_mongo_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
	$(DOCKER) build . $(DOCKER_BUILD_ARGS) --file $(DOCKERFILE_DIR)/Dockerfile-mongo --tag $(MONGO_IMG):$(VERSION) --tag $(MONGO_IMG):latest
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mongo --tag $(MONGO_IMG):latest
else
	$(DOCKER) buildx build .  $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mongo --tag $(MONGO_IMG):$(VERSION)
endif
endif

push_mongo_image: $(CMD_FILES) $(PKG_FILES)
ifneq ($(BUILDX_ENABLED), true)
ifeq ($(TAG_LATEST), true)
	$(DOCKER) push $(MONGO_IMG):latest
else
	$(DOCKER) push $(MONGO_IMG):$(VERSION)
endif
else
ifeq ($(TAG_LATEST), true)
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mongo --tag $(MONGO_IMG):latest --push
else
	$(DOCKER) buildx build . $(DOCKER_BUILD_ARGS) --platform $(BUILDX_PLATFORMS) --file $(DOCKERFILE_DIR)/Dockerfile-mongo --tag $(MONGO_IMG):$(VERSION) --push
endif
endif