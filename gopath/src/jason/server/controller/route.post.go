package controller

import (
	"github.com/kataras/iris/context"
)

func httpRoutePost(ctx context.Context) {
	ctx.HTML("<h1> Home </h1>")
	// this will print an error,
	// this route's handler will never be executed because the middleware's criteria not passed.
}
