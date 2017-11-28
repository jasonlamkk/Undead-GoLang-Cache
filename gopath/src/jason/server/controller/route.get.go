package controller

import (
	"encoding/json"
	"io"
	"jason/server/model"
	"net/http"
	"time"

	"github.com/kataras/iris/context"
)

const (
	PathGetRoutePrefix = "/route/"
)

var (
	freqUsedOutputNotFound   []byte
	freqUsedOutputInProgress []byte
)

func init() {
	freqUsedOutputNotFound, _ = json.Marshal(map[string]string{
		"status": "failure",
		"error":  "not found",
	})
	freqUsedOutputInProgress, _ = json.Marshal(map[string]string{
		"status": "in progress",
	})

}

//HTTPRouteGet controller for /route/{token}
func HTTPRouteGet(ctx context.Context) {
	var token = ctx.Params().Get("token")

	// fmt.Println("check token", token)
	localResult, peerAddress, isLocal, isReady, isPeer := model.GetRouteByToken(token)

	// fmt.Println(">>>local?", isLocal, ">>>ready?", isReady, ">>>", localResult)

	// fmt.Println(">>>peer?", isPeer, peerAddress)

	if !isLocal {
		if !isPeer || peerAddress == "" {
			ctx.StatusCode(200)
			ctx.Write(freqUsedOutputNotFound)
			return
		}
		// fmt.Println("redrect to", peerAddress)
		// ctx.Redirect("//"+peerAddress+PathGetRoutePrefix+token, 301)
		//CORRECTION, follow API give status 200; AJAX / XHTTPRequest may not handle redirect correctly

		//proxy
		oldReq := ctx.Request()
		// req.Host = peerAddress
		// req.RequestURI = PathGetRoutePrefix + token
		req, _ := http.NewRequest("GET", "http://"+peerAddress+PathGetRoutePrefix+token, nil)
		req.Header = oldReq.Header

		// fmt.Println("proxy", "http://"+peerAddress+PathGetRoutePrefix+token, req.Header)

		var client = http.Client{Timeout: time.Second}
		resp, err := client.Do(req)
		if err != nil {
			ctx.StatusCode(500)
			ctx.Write([]byte(err.Error()))
			return
		}
		ctx.StatusCode(200)
		for k := range resp.Header {
			ctx.Header(k, resp.Header.Get(k))
		}

		ctx.StreamWriter(func(w io.Writer) bool {
			io.Copy(w, resp.Body)
			resp.Body.Close()
			return false
		})
		return
	}

	if !isReady {
		ctx.StatusCode(200)
		ctx.Write(freqUsedOutputInProgress)
		return
	}

	ctx.StatusCode(200)
	ctx.Write(localResult)
	// done
}
