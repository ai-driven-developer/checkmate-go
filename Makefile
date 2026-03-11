.PHONY: build build-nnue build-simd test bench perft clean

BINARY = checkmatego

build:
	go build -o $(BINARY) ./cmd/checkmatego/

build-nnue:
	go build -tags embed_nnue -o $(BINARY) ./cmd/checkmatego/

build-simd:
	go build -tags simd -o $(BINARY) ./cmd/checkmatego/

nnue-simd:
	go build -tags 'embed_nnue simd' -o $(BINARY) ./cmd/checkmatego/

test:
	go test ./internal/... -v -timeout 120s

bench:
	go test ./internal/movegen/ -bench=BenchmarkPerft -benchtime=5s

perft: build
	echo -e "position startpos\nperft 6\nquit" | ./$(BINARY)

clean:
	rm -f $(BINARY)
