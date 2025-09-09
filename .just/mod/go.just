gen:
    go tool devtool gen --all ./internal/cmd/crkit

update:
    go get -u ./...

dep:
    go mod tidy

test:
    go test -failfast -count=1 ./...

test-race:
    CGO_ENABLED=1 go test -race -count=1 ./...

fmt:
    go tool gofumpt -w -l .
