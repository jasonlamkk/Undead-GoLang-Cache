package controller

func makeErrorResponse(err error) map[string]interface{} {
	return map[string]interface{}{
		"err": err.Error(),
	}
}
