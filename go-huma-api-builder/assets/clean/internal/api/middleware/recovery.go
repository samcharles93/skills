package middleware

import (
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func Recovery(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered: %v", r)
				huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next(ctx)
	}
}
