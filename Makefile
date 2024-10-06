BIN = dynamic-dns-service
SOURCES = $(shell find . -name '*.go')

all: test $(BIN)

$(BIN): $(SOURCES)
	cd cmd/$(BIN) && \
	go build -o $(BIN) .

.PHONY: clean
clean:
	rm -f $(BIN)

.PHONY: test
test: $(SOURCES)
	go test -v ./...

.PHONY: run
run: $(BIN)
	./$(BIN)
