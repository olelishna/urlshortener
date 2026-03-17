package config

import (
	"flag"
	"os"
)

var (
	FlagRunAddr       string
	FlagBaseURLResult string
)

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(
		&FlagBaseURLResult,
		"b",
		"http://localhost:8080",
		"base address for shortened url",
	)
	flag.Parse()

	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		FlagRunAddr = envRunAddr
	}

	if envBaseURLResult := os.Getenv("BASE_URL"); envBaseURLResult != "" {
		FlagBaseURLResult = envBaseURLResult
	}
}
