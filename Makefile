BIN = dynamic-dns-service
SOURCES = $(shell find . -name '*.go')

all: $(BIN)

build: $(BIN)

$(BIN): $(SOURCES)
	go build -o bin/$(BIN) ./...

.PHONY: clean
clean:
	rm -f bin/$(BIN)

.PHONY: test
test: $(SOURCES)
	go test -v ./...

.PHONY: run
run: $(BIN)
	./$(BIN)

.PHONY: deploy
deploy: $(BIN)
	ansible-playbook -i deployments/playbooks/inventory.ini deployments/playbooks/main.yaml
