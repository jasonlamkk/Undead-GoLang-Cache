package model

import (
	"context"
	"fmt"
	"time"

	"googlemaps.github.io/maps"
)

var (
	mapClient *maps.Client
)

//InitGoogleMapWithApiKey pass api key and create API client
func InitGoogleMapWithApiKey(key string) {
	var err error
	mapClient, err = maps.NewClient(maps.WithAPIKey(key), maps.WithRateLimit(2))
	if err != nil {
		fmt.Println("Fail to init map client", err)
		panic("ASSERT - no map client, no need to continue")
	}
}

// func getTrafficModelByTimezone()

func processRouteRequest(ctx context.Context, id string, payload interface{}) {

	fmt.Println("process token", id, "with payload", payload)
	if mapClient == nil {
		panic("ASSERT - map client not init and API key not set")

		//only run this if above line removed
		cancelRouteRequest(id)
		return
	}

	mrData, ok := payload.([][]string)
	if !ok {
		fmt.Println("[ERROR]", "processRouteRequest", "invalid payload")
		cancelRouteRequest(id)
		return
	}

	inputLen := len(mrData)
	if inputLen < 2 || len(mrData[0]) != 2 {
		fmt.Println("[ERROR]", "processRouteRequest", "invalid payload")
		cancelRouteRequest(id)
		return
	}

	//build request
	var origin string = mrData[0][0] + "," + mrData[0][1]
	var waypoints []string = make([]string, inputLen-2)
	var dest string = mrData[inputLen-1][0] + "," + mrData[inputLen-1][1]

	for i := 1; i < inputLen-1; i++ {
		if len(mrData[i]) < 2 {
			fmt.Println("[ERROR]", "processRouteRequest", "invalid destination")
			cancelRouteRequest(id)
			return
		}
		waypoints[i-1] = mrData[i][0] + "," + mrData[i][1]
	}

	var req = &maps.DirectionsRequest{
		Alternatives:  false,
		Avoid:         []maps.Avoid{maps.AvoidFerries},
		DepartureTime: "now",
		Language:      "en",
		Origin:        origin,
		Mode:          maps.TravelModeDriving,
		Units:         maps.UnitsMetric,
		TrafficModel:  maps.TrafficModelOptimistic,
		Destination:   dest,
	}
	if len(waypoints) > 0 {
		fmt.Println("wp added:", waypoints)
		req.Waypoints = waypoints
	} else {
		fmt.Println("wp?", waypoints)
	}
	// var req = &maps.DistanceMatrixRequest{
	// 	Origins:       []string{origin},
	// 	Destinations:  dests,
	// 	Mode:          maps.TravelModeDriving,
	// 	Language:      "",
	// 	Avoid:         maps.AvoidFerries,
	// 	Units:         maps.UnitsMetric,
	// 	TrafficModel:  maps.TrafficModelOptimistic,
	// 	DepartureTime: "now",
	// }
	fmt.Println("send to google map", req)
	timeoutContext, cancelCleanUp := context.WithTimeout(ctx, time.Second*5)
	defer cancelCleanUp()

	routes, _, err := mapClient.Directions(timeoutContext, req)
	if err != nil {
		fmt.Println("request failed", err)
	} else if len(routes) < 1 {
		fmt.Println("result empty")
	} else {
		builder := newRouteResultSuccessBuilder()
		for _, leg := range routes[0].Legs {
			sl := leg.StartLocation
			el := leg.EndLocation
			fmt.Println("Leg", leg.StartLocation, leg.EndLocation, leg.HumanReadable)
			builder.addLegStat(sl.Lat, sl.Lng, el.Lat, el.Lng, leg.Duration, leg.Meters)
		}

		getRouteRequestStore().putResult(id, builder.getJSON())
		// fmt.Println("done", string(builder.getJSON()))
		return
	}

	cancelRouteRequest(id)
	// // fmt.Println(">>>>")

	// // for i, r := range routes {
	// // 	fmt.Println(">>", i, ">>", r)
	// // 	fmt.Println(r.Text)
	// // 	r.Legs[0].Duration
	// // }

	// // for i, r := range routes {
	// // 	fmt.Println(">>", i, ">>", r)
	// // 	fmt.Println(r.Text)
	// // }
	// // for a, b := range resp.Rows {
	// // 	fmt.Println("Rows:", a, b)
	// // 	for _, e := range b.Elements {
	// // 		fmt.Println(">>>", e.Status, ">Distance>", e.Distance, ">Duration>", e.Duration, ">DurationInTraffic>", e.DurationInTraffic)
	// // 	}
	// // }
	// // }

	// // fmt.Println("Google Map Result", resp, err)
	// cancelCleanUp()
}

func cancelRouteRequest(id string) {
	fmt.Println("give up process token", id)
	getRouteRequestStore().putResult(id, buildRouteResultError("upstream operation timeout"))
}
