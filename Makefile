clean:
	rm -rf src/proto/gen

build:
	mkdir -p src/proto/gen
	protoc \
		-I=./src/proto \
		--go_out=./src/proto/gen \
		--go-grpc_out=./src/proto/gen \
		 src/proto/*.proto