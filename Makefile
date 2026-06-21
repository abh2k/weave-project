PROTO_DIR := proto
PROTO_FILES := proto/weave/api/v1/*.proto

.PHONY: proto-gen run-grpc

proto-gen:
	mkdir -p gen/proto
	protoc -I $(PROTO_DIR) \
		--go_out=gen/proto --go_opt=paths=source_relative \
		--go-grpc_out=gen/proto --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)

run-grpc:
	go run ./cmd/api
