package model

import (
	"encoding/json"
	"fmt"
	"time"
)

//object keep private, field public (visible to json encoder)
type routeResult struct {
	Status        string     `json:"status"`
	Path          [][]string `json:"path"`
	TotalDistance int        `json:"total_distance"`
	TotalTime     int        `json:"total_time"`
}

type routeResultBuilder struct {
	obj routeResult
}

func newRouteResultSuccessBuilder() *routeResultBuilder {
	return &routeResultBuilder{
		routeResult{
			Status:        "success",
			Path:          nil,
			TotalDistance: 0,
			TotalTime:     0,
		},
	}
}
func wrapLatLng(lat, lng float64) []string {
	return []string{fmt.Sprint(lat), fmt.Sprint(lng)}
}

//this code shall not depend on structs from google maps API
func (b *routeResultBuilder) addLegStat(slat, slng, elat, elng float64, dur time.Duration, meter int) {
	if nil == (b.obj.Path) {
		b.obj.Path = [][]string{wrapLatLng(slat, slng)}
	}
	b.obj.Path = append(b.obj.Path, wrapLatLng(elat, elng))
	b.obj.TotalDistance += meter
	b.obj.TotalTime += int(dur.Seconds())
}

func (b *routeResultBuilder) getJSON() (buf []byte) {
	buf, _ = json.Marshal(b.obj) //always success
	return buf
}

func buildRouteResultError(err string) (buf []byte) {
	buf, _ = json.Marshal(map[string]string{
		"status": "failure",
		"error":  err,
	}) //always success
	return buf
}
