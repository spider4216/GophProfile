APP_NAME=gophkeeper

DSN?=postgres://postgres:postgres@localhost:5432/gophprofile
LOG_LEVEL?=debug
SERVER_HOST?=https://127.0.0.1:8080

run-srv:
	SERVER_ADDRESS=${SERVER_HOST} DB_DSN=$(DSN) LOG_LEVEL=${LOG_LEVEL} go run ./cmd/server

crt:
	go run ./cmd/crt