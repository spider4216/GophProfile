APP_NAME=gophkeeper

DSN?=postgres://postgres:postgres@localhost:5432/gophprofile
LOG_LEVEL?=debug
SERVER_HOST?=127.0.0.1:8080

run-srv:
	SERVER_ADDRESS=${SERVER_HOST} DB_DSN=$(DSN) LOG_LEVEL=${LOG_LEVEL} go run ./cmd/server

crt:
	go run ./cmd/crt

migration-gen:
	migrate create -ext sql -dir ./migrations -seq $(name)

migrate-up:
	migrate -path ./migrations -database $(DSN) up $(ver)

migrate-down:
	migrate -path ./migrations -database $(DSN) down $(ver)

migrate-force:
	migrate -path ./migrations -database $(DSN) force $(ver)