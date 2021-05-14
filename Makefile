.PHONY: cover
cover:
	go test -coverprofile=bin/coverage.out
	go tool cover -html=bin/coverage.out
