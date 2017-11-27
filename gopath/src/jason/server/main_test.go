package main_test

import "testing"

func TestMain(m *testing.M) {

	// build := exec.Command("go build .")
	// _, err := build.CombinedOutput()
	// if err != nil {
	// 	panic(err)
	// }

	m.Run()
}

func runWithConfig(c string) {
}
