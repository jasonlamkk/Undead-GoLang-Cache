package controller

import (
	"github.com/kataras/iris/context"

	"jason/server/model"
)

//HttpRoutePost controller for /route
func HttpRoutePost(ctx context.Context) {
	var input = make([][]string, 0)
	if err := ctx.ReadJSON(&input); err != nil {
		//always http 200
		ctx.JSON(makeErrorResponse(err))
		return
	}

	token, err := model.RegisterRouteRequestAsync(input)
	if err != nil {
		ctx.JSON(makeErrorResponse(err))
		return
	}
	ctx.JSON(map[string]string{"token": token})
}
