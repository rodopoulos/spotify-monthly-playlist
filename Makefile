clean:
	@rm .spotify-token
	@rm srodofy

build:
	@go build src/* -o srodofy

run:
	@go run ./...
