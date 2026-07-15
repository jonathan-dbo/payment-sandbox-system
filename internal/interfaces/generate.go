package interfaces

//go:generate oapi-codegen -config ../api/cfg.yaml ../api/api.yaml

// backup here go:generate oapi-codegen -generate types,strict-server,std-http-server -package handlers -o internal/interfaces/handlers/oapi_gen.go api.yaml
