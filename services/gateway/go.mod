module github.com/Riku-KANO/ec-test/services/gateway

go 1.25.3

require (
	github.com/Riku-KANO/ec-test/pkg v0.0.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
)

replace github.com/Riku-KANO/ec-test/pkg => ../../pkg
