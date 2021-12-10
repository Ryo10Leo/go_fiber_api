.PHONY: create-user create-product create-order update-ranking up

create-user:
	docker-compose run --rm backend sh -c "go run src/commands/populateUsers.go"

create-product:
	docker-compose run --rm backend sh -c "go run src/commands/product/populateProducts.go"

create-order:
	docker-compose run --rm backend sh -c "go run src/commands/order/populateOrders.go"

update-ranking:
	docker-compose run --rm backend sh -c "go run src/commands/redis/updateRankings.go"
	
up:
	docker-compose up