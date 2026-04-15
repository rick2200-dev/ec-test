module github.com/Riku-KANO/ec-test/services/inventory/api/gen/go

go 1.25.3

require (
	github.com/Riku-KANO/ec-test/shared/api/gen/go v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260311181403-84a4fc48630c // indirect
)

replace github.com/Riku-KANO/ec-test/shared/api/gen/go => ../../../../../shared/api/gen/go
