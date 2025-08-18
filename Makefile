simple-test-1:
	nc -u localhost 2053

simple-test-2:
	dig @127.0.0.1 -p 2053 example.com

run:
	air .

## TODO: See if I can work this with air.
run-with-forwarding:
	go run cmd/server/main.go --resolver=1.1.1.1:53

test:
	go test $$(go list ./... | grep -v '/cmd/') -race -cover --coverprofile=coverage.out -covermode=atomic