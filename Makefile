run:
	docker-compose up --build -d app postgres

down:
	docker-compose down -v

test.unit:
	go test ./... -coverprofile=coverage.out -coverpkg=./... -cover -v | \
		grep -v 'github.com/l-golofastov/Merch-Shop-Api/internal/handlers/auth/mocks/' | \
		grep -v 'github.com/l-golofastov/Merch-Shop-Api/internal/handlers/buy/mocks/' | \
		grep -v 'github.com/l-golofastov/Merch-Shop-Api/internal/handlers/info/mocks/' | \
		grep -v 'github.com/l-golofastov/Merch-Shop-Api/internal/handlers/sendCoins/mocks/'

	go tool cover -func=coverage.out

test.integration:
	docker-compose up -d postgres-test
	while ! docker-compose exec postgres-test pg_isready -U ${TEST_POSTGRES_USERNAME} -d ${TEST_POSTGRES_DBNAME}; do sleep 1; done

	-go test -v ./tests

	make down

test:
	make test.unit
	make test.integration
