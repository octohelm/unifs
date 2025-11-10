devtool := 'go tool devtool'

gen:
    {{ devtool }} gen --all ./cmd/unifs

update:
    go get -u ./...

dep:
    go mod tidy

test:
    go test -failfast -count=1 ./...

test-race:
    CGO_ENABLED=1 go test -race -count=1 ./...

fmt:
    {{ devtool }} fmt -w -l .
