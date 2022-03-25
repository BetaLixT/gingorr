package gingorr

import (
	"github.com/betalixt/gorr"
	"github.com/gin-gonic/gin"
)

func handleErrors(ctx *gin.Context) {

	ctx.Next()
	lerr := ctx.Errors.Last()
	if lerr != nil {
		pst, ok := lerr.Err.(*gorr.Error)
		if ok {
			ctx.JSON(pst.StatusCode, pst)
		} else {
			ctx.JSON(500, gorr.UnexpectedError(lerr))
		}
	}
}
