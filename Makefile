test:
	go test ./...
	go test -race ./...
	go test -tags=integration ./...
	# go test -cover -tags=integration ./...

.PHONY: test
