PROBLEM_SERVICE_NAME=problem-service

.PHONY: build
build:
	@echo "Building Problem Management Service with docker-compose..."
	docker compose build $(PROBLEM_SERVICE_NAME)

.PHONY: run
run: run-problem-service

.PHONY: run-problem-service
run-problem-service:
	@echo "Running Problem Management Service with docker-compose..."
	docker compose up -d $(PROBLEM_SERVICE_NAME)

.PHONY: stop
stop: stop-problem-service

.PHONY: stop-problem-service
stop-problem-service:
	@echo "Stopping and removing Problem Management Service..."
	docker compose down

.PHONY: clean
clean:
	@echo "Cleaning up Docker images..."
	docker compose down --rmi all
