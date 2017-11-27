package controller

import (
	"github.com/kataras/iris/context"

	"jason/server/cluster"
	"jason/server/model"
)

//HTTPRoutePost controller for /route
func HTTPRoutePost(ctx context.Context) {
	var input = make([][]string, 0)
	if err := ctx.ReadJSON(&input); err != nil {
		//always http 200
		ctx.JSON(makeErrorResponse(err))
		return
	}

	token, expire, err := model.RegisterRouteRequestAsync(input)
	if err != nil {
		ctx.JSON(makeErrorResponse(err))
		return
	}
	ctx.JSON(map[string]string{"token": token})

	go cluster.PublishRouteOwnership(token, expire)
}
