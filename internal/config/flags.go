package config

import (
	"flag"
	"os"
)

var (
	FlagRunAddr         string
	FlagBaseURLResult   string
	FlagLogLevel        string
	FlagFileStoragePath string
)

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(
		&FlagBaseURLResult,
		"b",
		"http://localhost:8080",
		"base address for shortened url",
	)
	flag.StringVar(&FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&FlagFileStoragePath, "f", "data/data.json", "file storage path")
	flag.Parse()

	if envRunAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		FlagRunAddr = envRunAddr
	}

	if envBaseURLResult, ok := os.LookupEnv("BASE_URL"); ok {
		FlagBaseURLResult = envBaseURLResult
	}

	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		FlagLogLevel = envLogLevel
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		FlagFileStoragePath = envFileStoragePath
	}
}
