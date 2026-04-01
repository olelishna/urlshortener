package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	_clientTimeOut       = 30 * time.Second
	_idleConnTimeout     = 90 * time.Second
	_maxIdleConns        = 100
	_maxIdleConnsPerHost = 10
)

func main() {
	endpoint := "http://localhost:8080/"

	fmt.Println("Введите длинный URL")

	reader := bufio.NewReader(os.Stdin)

	long, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	long = strings.TrimSuffix(long, "\n")

	client := &http.Client{
		Timeout: _clientTimeOut,
		Transport: &http.Transport{
			MaxIdleConns:        _maxIdleConns,
			MaxIdleConnsPerHost: _maxIdleConnsPerHost,
			IdleConnTimeout:     _idleConnTimeout,
		},
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(long))
	if err != nil {
		panic(err)
	}

	request.Header.Add("Content-Type", "text/plain")

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	fmt.Println("Статус-код ", response.Status)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
