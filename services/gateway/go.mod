module github.com/Riku-KANO/ec-test/services/gateway

go 1.25.3

require (
	github.com/Riku-KANO/ec-test/gen/go v0.0.0-00010101000000-000000000000
	github.com/Riku-KANO/ec-test/pkg v0.0.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.80.0
)

require (
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260311181403-84a4fc48630c // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/Riku-KANO/ec-test/pkg => ../../pkg

replace github.com/Riku-KANO/ec-test/gen/go => ../../gen/go
