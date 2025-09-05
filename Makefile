
fmt:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/daixiang0/gci@latest
	go fmt ./...
	goimports -w .
	gci -w .

clean:
	go clean -testcache

