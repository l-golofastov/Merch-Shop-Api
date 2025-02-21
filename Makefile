run:
	docker-compose up --build -d app postgres

down:
	docker-compose down -v

test.unit:
	go test -v ./... --short

test.integration:
	docker-compose up -d postgres-test
	while ! docker-compose exec postgres-test pg_isready -U ${TEST_POSTGRES_USERNAME} -d ${TEST_POSTGRES_DBNAME}; do sleep 1; done

	-go test -v ./tests

	make down

test:
	make test.unit
	make test.integration
