package middlewares

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	pbauth "github.com/PaperMan11/goim/pkg/protocol/auth"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/gin-gonic/gin"
)

var whiteList = []string{
	"/api/v1/auth/get_admin_token",
	"/api/v1/auth/parse_token",
}

func isWhiteList(path string) bool {
	for _, item := range whiteList {
		if item == path {
			return true
		}
	}
	return false
}

func Auth(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isWhiteList(c.Request.URL.Path) {
			c.Next()
			return
		}
		token := c.Request.Header.Get(constant.Token)
		if token == "" {
			apiresp.Error(c.Writer, errx.TokenNotExistError)
			c.Abort()
			return
		}
		resp, err := svc.AuthService.ParseToken(c, &pbauth.ParseTokenReq{Token: token})
		if err != nil {
			apiresp.Error(c.Writer, errx.TokenNotValidYetError.WrapWithError(err))
			c.Abort()
			return
		}
		c.Set(constant.OpUserPlatform, constant.PlatformIDToName(int(resp.PlatformID)))
		c.Set(constant.OpUserID, resp.UserID)
		c.Next()
	}
}

func ParseOperationID(c *gin.Context) {
	opID := c.Request.Header.Get(constant.OperationID)
	if opID == "" {
		apiresp.Error(c.Writer, errx.ArgsError.Wrap("operationID is empty"))
		c.Abort()
		return
	}
	c.Set(constant.OperationID, opID)
	c.Next()
}
