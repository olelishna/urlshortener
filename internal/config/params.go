package config

import "os"

var (
	AesKey = "521e97ce0ea76c4176f226b1a49022e64bc84c914e675655bb85b7e08d3fac88" // default sample
)

func GetEnvParams() {
	if envAESKey, ok := os.LookupEnv("AES_KEY"); ok {
		AesKey = envAESKey
	}
}
