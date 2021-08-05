REPO_ROOT := $(shell git rev-parse --show-toplevel)
UNIT_TEST_DIR := $(REPO_ROOT)/pkg/eraser

run:
	cd $(UNIT_TEST_DIR); go test ./... -coverprofile cover.out
