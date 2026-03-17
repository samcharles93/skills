package middleware

import (
	"log"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func Logging() func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		start := time.Now()
		method := ctx.Method()
		path := ctx.URL().Path

		next(ctx)

		status := ctx.Status()
		duration := time.Since(start)

		log.Printf("%s %s %d %v", method, path, status, duration)
	}
}
