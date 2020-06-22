package main

import (
	"datamon/dbcheck"
	"datamon/metrics"
	"fmt"
)

func main() {
	fmt.Println("Start App")
	go dbcheck.Start()
	metrics.Start()
}
