BIN_DIR := bin
SERVICES := rest soap sftp

all: $(BIN_DIR) $(SERVICES)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(BIN_DIR)/%: cmd/%
	go build -o $@ ./cmd/$*

rest: $(BIN_DIR)/rest
soap: $(BIN_DIR)/soap
sftp: $(BIN_DIR)/sftp

clean:
	rm -rf $(BIN_DIR)

run: all
	go run orchestrator/main.go
