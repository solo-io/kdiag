package api

//go:generate protoc --proto_path=../../api --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative kdiag/api.proto
