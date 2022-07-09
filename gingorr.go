package gingorr

import (
	"errors"
	"fmt"
	"net"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/betalixt/gorr"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ErrorHandlerMiddleware(spkey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tctx := ctx.MustGet(spkey).(IServiceProvider)
		ctx.Next()
		lgr := tctx.GetLogger()

		if len(ctx.Errors) > 0 {
			errs := make([]error, len(ctx.Errors))
			berr := (*gorr.Error)(nil)
			var temp *gorr.Error
			for idx, err := range ctx.Errors {
				errs[idx] = err.Err
				if berr != nil && errors.As(err.Err, &temp) {
					berr = temp
				}
			}
			lgr.Error("errors processing request", zap.Errors("error", errs))
			if berr != nil {
				ctx.JSON(berr.StatusCode, berr)
			} else {
				ctx.JSON(500, gorr.NewUnexpectedError(ctx.Errors[len(ctx.Errors)-1]))
			}
		} else {
			if !ctx.Writer.Written() {
				lgr.Error("No response was written")
				ctx.JSON(500, gorr.NewError(
					gorr.ErrorCode{
						Code: 11001,
						Message: "UnsetResponse",
					},
					500,
					"",
				))
			}
		}
	}
}

func RecoveryMiddleware(
	spkey string,
	logger *zap.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		tctxAny, exists := c.Get(spkey) 
		defer func() {
			if err := recover(); err != nil {
				
				// Dependent on the txgenerator
				// Making it more resilient to avoid errors
				lgr := (*zap.Logger)(nil)
				if !exists {
					lgr = logger
				} else if tctx, ok := tctxAny.(IServiceProvider); !ok {
					lgr = logger
				} else {
					lgr = tctx.GetLogger()
				}

				// In case the get logger fails
				if lgr == nil {
					lgr = logger
				}	
				
				perr, ok := err.(gorr.Error)
				if ok {
					c.JSON(perr.StatusCode, perr)
				} else {
					// Check for a broken connection, as it is not really a
					// condition that warrants a panic stack trace.
					var brokenPipe bool
					if ne, ok := err.(*net.OpError); ok {
						var se *os.SyscallError
						if errors.As(ne, &se) {
							if strings.Contains(
								strings.ToLower(se.Error()), "broken pipe") ||
								strings.Contains(strings.ToLower(se.Error()),
								"connection reset by peer",
								) {
								brokenPipe = true
							}
						}
					}
					
					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					headers := strings.Split(string(httpRequest), "\r\n")
					for idx, header := range headers {
						current := strings.Split(header, ":")
						if current[0] == "Authorization" {
							headers[idx] = current[0] + ": *"
						}
					}
					headersToStr := strings.Join(headers, "\r\n")
					if brokenPipe {	
						lgr.Error(
							"Panic recovered, broken pipe",
							zap.String("headers", headersToStr),
							zap.Any("error", err),
						)
						c.Abort()
					} else {	
						lgr.Error(
							"Panic recovered",
							zap.String("headers", headersToStr),
							zap.Any("error", err),
							zap.Stack("stack"),
						)
						c.JSON(500, gorr.NewUnexpectedError(fmt.Errorf("%v", err)))
					}
				}
			}
		}()
		c.Next()
	}
}

// Root level panic handler with minimal dependencies
func RootRecoveryMiddleware(lgr *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				lgr.Error(
					"panic reached root handling (this is bad)",
					zap.Any("error", err),
					zap.Stack("stack"),
				)
				c.JSON(500, gin.H{
					"errorCode": 10000,
					"errorMessage": "UnexpectedError",
					"errorDetail": "panic reached root",
				})
			}
		}()
		c.Next()
	}
}
