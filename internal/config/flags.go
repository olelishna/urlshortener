package config

import (
	"flag"
)

var FlagRunAddr string
var FlagBaseUrlResult string

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&FlagBaseUrlResult, "b", "http://localhost:8080/", "base address for shortened url")
	flag.Parse()
}
