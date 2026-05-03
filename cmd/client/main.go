package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/olelishna/urlshortener/internal/logger"
	"go.uber.org/zap"
)

const (
	ClientTimeOut       = 30 * time.Second
	IdleConnTimeout     = 90 * time.Second
	MaxIdleConns        = 100
	MaxIdleConnsPerHost = 10
)

func main() {
	endpoint := "http://localhost:8080/"

	fmt.Println("Введите длинный URL")

	reader := bufio.NewReader(os.Stdin)

	long, err := reader.ReadString('\n')
	if err != nil {
		logger.Log.Fatal(err.Error(), zap.String("event", "read long"))
	}

	long = strings.TrimSuffix(long, "\n")

	client := &http.Client{
		Timeout: ClientTimeOut,
		Transport: &http.Transport{
			MaxIdleConns:        MaxIdleConns,
			MaxIdleConnsPerHost: MaxIdleConnsPerHost,
			IdleConnTimeout:     IdleConnTimeout,
		},
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(long))
	if err != nil {
		logger.Log.Fatal(err.Error(), zap.String("event", "create request"))
	}

	request.Header.Add("Content-Type", "text/plain")

	response, err := client.Do(request)
	if err != nil {
		logger.Log.Fatal(err.Error(), zap.String("event", "do request"))
	}

	fmt.Println("Статус-код ", response.Status)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Log.Fatal(err.Error(), zap.String("event", "read response"))
	}

	fmt.Println(string(body))
}
