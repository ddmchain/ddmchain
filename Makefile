
.PHONY: gddm clean
.PHONY: gddm-linux gddm-linux-386 gddm-linux-amd64


GOBIN = $(shell pwd)/release
GO ?= latest

gddm:
	path/env.sh go run path/ci.go install ./ctrl/gddm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gddm\" to launch gddm."

clean:
	rm -fr path/_workspace/pkg/ $(GOBIN)/*


devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'
