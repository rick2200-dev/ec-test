module github.com/Riku-KANO/ec-test/services/order

go 1.25.3

replace github.com/Riku-KANO/ec-test/pkg => ../../pkg

replace github.com/Riku-KANO/ec-test/gen/go => ../../gen/go

require (
	github.com/Riku-KANO/ec-test/gen/go v0.0.0-00010101000000-000000000000
	github.com/Riku-KANO/ec-test/pkg v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.1
	github.com/stripe/stripe-go/v82 v82.5.1
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260311181403-84a4fc48630c // indirect
)
