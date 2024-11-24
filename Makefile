all: clean build

clean:
	rm -rf src/proto

build:
	mkdir -p src/proto
	protoc \
		-I=. \
		--go_out=./src/proto \
		--go-grpc_out=./src/proto \
		 proto/*.proto