test: 
	go vet -shadow && GOCACHE=off go test -v -race ./...
