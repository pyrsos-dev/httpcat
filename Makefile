BIN := httpcat

$(BIN): main.go
	go build -o $(BIN) main.go
