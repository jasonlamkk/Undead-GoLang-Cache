package controller

import (
	"github.com/kataras/iris"
)

//AddAllRoute is the entry point for all controller routes, which shall be package private
func AddAllRoute(app *iris.Application) {

	app.Post("/route", httpRoutePost)

	// app.Get("/route/{token:string regexp()}", httpRouteGet)
}
