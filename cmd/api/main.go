package main

import (
	"fmt"

	"github.com/gonszalito/go-ddd-architecture/internal/config"
	infraHTTP "github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http"
	"github.com/gonszalito/go-ddd-architecture/internal/wire"
)

func main() {
	app, cleanup, err := wire.InitializeApp()
	if err != nil {
		panic(err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	cfg := config.MustLoad()
	r := infraHTTP.NewRouterCodegen(infraHTTP.RouterDependencies{
		UserService:        app.UserService,
		InvoiceService:     app.InvoiceService,
		PaymentService:     app.PaymentService,
		RefundService:      app.RefundService,
		TopUpService:       app.TopUpService,
		Dashboard:          app.Dashboard,
		JWTSecret:          cfg.JWTSecret,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		EnableDocs:         true,
		SwaggerURL:         "http://localhost:8080/api.yaml",
	})

	r.Run(fmt.Sprintf(":%s", cfg.Port))
}
