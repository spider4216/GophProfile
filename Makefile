APP_NAME=gophkeeper

DSN?=postgres://postgres:postgres@localhost:5432/gophprofile
LOG_LEVEL?=debug
SERVER_HOST?=127.0.0.1:8080
RABBIT_DSN?=amqp://guest:guest@localhost:5672/
MINIO_HOST?=http://localhost:9000
MINIO_ROOT_USER?=minio
MINIO_ROOT_PASSWORD?=minio123


run-srv:
	MINIO_HOST=${MINIO_HOST} MINIO_ROOT_USER=${MINIO_ROOT_USER} MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD} RABBIT_DSN=$(RABBIT_DSN) SERVER_ADDRESS=${SERVER_HOST} DB_DSN=$(DSN) LOG_LEVEL=${LOG_LEVEL} go run ./cmd/server

run-worker:
	MINIO_ROOT_USER=${MINIO_ROOT_USER} MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD} DB_DSN=$(DSN) MINIO_HOST=${MINIO_HOST} RABBIT_DSN=$(RABBIT_DSN) LOG_LEVEL=${LOG_LEVEL} go run ./cmd/worker

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