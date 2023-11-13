package mid

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/shawnzxx/service/foundation/web"
	"go.uber.org/zap"
)

// this logger at business layer, becuase as top app layer's middleware it can log request and response in general
// later business layer logic also can use this logger to log business realted logs
func Logger(log *zap.SugaredLogger) web.Middleware {
	m := func(handler web.Handler) web.Handler {

		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			v := web.GetValues(ctx)

			path := r.URL.Path
			if r.URL.RawQuery != "" {
				path = fmt.Sprintf("%s?%s", path, r.URL.RawQuery)
			}

			log.Infow("request started", "trace_id", v.TraceID, "method", r.Method, "path", path,
				"remoteaddr", r.RemoteAddr)

			err := handler(ctx, w, r)

			log.Infow("request completed", "trace_id", v.TraceID, "method", r.Method, "path", path,
				"remoteaddr", r.RemoteAddr, "statuscode", v.StatusCode, "since", time.Since(v.Now))

			return err
		}

		return h
	}

	return m
}
