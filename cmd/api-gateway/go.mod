module github.com/quantumlayer-factory-hq/quantumlayer-factory/cmd/api-gateway

go 1.23.1

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/quantumlayer-factory-hq/quantumlayer-factory v0.0.0
	github.com/rs/cors v1.11.1
	go.temporal.io/sdk v1.36.0
)

replace github.com/quantumlayer-factory-hq/quantumlayer-factory => ../../
