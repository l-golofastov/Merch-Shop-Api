ifneq (,$(wildcard ./.env))
	include .env
	export
endif

test:
	go test --short -coverprofile=cover.out -v ./...
	make test.coverage

test.coverage:
	go tool cover -func=cover.out

test.integration:
	docker run --name=testPostgresContainer -p 5556:5432 -e POSTGRES_USER=$(TEST_POSTGRES_USERNAME) -e POSTGRES_PASSWORD=$(TEST_POSTGRES_PASSWORD) -e POSTGRES_DB=$(TEST_POSTGRES_DBNAME) -d --rm postgres

	-go test -v ./tests/

	docker stop testPostgresContainer