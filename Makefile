.PHONY: build build-nosimd test bench perft clean

BINARY = checkmatego

build:
	go build -o $(BINARY) ./cmd/checkmatego/

build-nosimd:
	go build -tags nosimd -o $(BINARY) ./cmd/checkmatego/

test:
	go test ./internal/... -v -timeout 120s

bench:
	go test ./internal/movegen/ -bench=BenchmarkPerft -benchtime=5s

perft: build
	echo -e "position startpos\nperft 6\nquit" | ./$(BINARY)

clean:
	rm -f $(BINARY)
