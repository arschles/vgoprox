package actions

import (
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/packr"
)

var proxy *render.Engine
var assetsBox = packr.NewBox("../public")

func init() {
	proxy = render.New(render.Options{
		// HTML layout to be used for all HTML requests:
		HTMLLayout:       "application.html",
		JavaScriptLayout: "application.js",

		// Box containing all of the templates:
		TemplatesBox: packr.NewBox("../templates/proxy"),
		AssetsBox:    assetsBox,

		// Add template helpers here:
		Helpers: render.Helpers{},
	})
}
