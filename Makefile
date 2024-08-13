test:
	go test -cover -race -count 1

benchmark:
	go test -bench . -benchmem
