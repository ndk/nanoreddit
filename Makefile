.PHONY: test
service.up:
	docker-compose -f docker-compose.infra.yaml -f docker-compose.service.yaml up --build --abort-on-container-exit
	docker-compose -f docker-compose.infra.yaml -f docker-compose.service.yaml down --volumes

infra.up:
	docker-compose -f docker-compose.infra.yaml up --build --abort-on-container-exit
