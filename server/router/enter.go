package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	jxcrouter "github.com/flipped-aurora/gin-vue-admin/server/router/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/router/media"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System  system.RouterGroup
	Example example.RouterGroup
	Media   media.RouterGroup
	Jxc     jxcrouter.RouterGroup
}
