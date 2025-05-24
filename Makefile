.PHONY: build run dev stop clean

build:
	docker build -t sentinel .

dev: build
	docker-compose -f docker-compose.dev.yml up --build --force-recreate

dev-logs: build
	docker-compose -f docker-compose.dev.yml up --build --force-recreate

dev-detach: build
	docker-compose -f docker-compose.dev.yml up -d --build --force-recreate

stop:
	docker-compose -f docker-compose.dev.yml down
	docker stop sentinel || true
	docker rm sentinel || true

test:
	docker-compose -f docker-compose.test.yml build --no-cache
	docker-compose -f docker-compose.test.yml up --force-recreate


clean: stop
	docker rmi sentinel || true