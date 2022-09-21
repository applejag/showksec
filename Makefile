# SPDX-FileCopyrightText: 2022 Kalle Fagerberg
# SPDX-License-Identifier: CC0-1.0

ifeq ($(OS),Windows_NT)
OUT_FILE = showksec.exe
else
OUT_FILE = showksec
endif

GO_FILES = $(shell git ls-files "*.go")

.PHONY: all
all: build

.PHONY: build
build: $(OUT_FILE)

$(OUT_FILE): $(GO_FILES)
	go build -o $(OUT_FILE) -ldflags='-s -w'

.PHONY: install
install:
	go install -ldflags='-s -w'

.PHONY: clean
clean:
	rm -rfv ./dinkur.exe ./dinkur

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: deps
deps: deps-go deps-pip deps-npm

.PHONY: deps-go
deps-go:
	go install github.com/mgechev/revive@latest
	go install golang.org/x/tools/cmd/goimports@latest

.PHONY: deps-pip
deps-pip:
	python3 -m pip install --upgrade --user reuse

.PHONY: deps-npm
deps-npm: node_modules

node_modules:
	npm install

.PHONY: lint
lint: lint-md lint-go lint-license

.PHONY: lint-fix
lint-fix: lint-md-fix lint-go-fix

.PHONY: lint-md
lint-md:
	npx remark .

.PHONY: lint-md-fix
lint-md-fix:
	npx remark . -o

.PHONY: lint-go
lint-go:
	@echo goimports -d '**/*.go'
	@goimports -d $(GO_FILES)
	revive -formatter stylish -config revive.toml ./...

.PHONY: lint-go-fix
lint-fix-go:
	@echo goimports -d -w '**/*.go'
	@goimports -d -w $(GO_FILES)

.PHONY: lint-license
lint-license:
	reuse lint
