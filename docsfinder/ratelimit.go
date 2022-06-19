package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func checkRate(flags Flags) (RateLimit, error) {
	url := fmt.Sprintf("https://api.github.com/rate_limit")
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("token %s", flags.Token)},
	}

	if flags.IncludePrivate {
		req.Header = headers
	}

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal(fmt.Sprintf("Statuscode: %s, %s", strconv.Itoa(res.StatusCode), url))
	}

	responseData, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Fatal(err)
	}

	var rate RateLimit

	if err := json.Unmarshal(responseData, &rate); err != nil {
		fmt.Println("failed to unmarshal:", err)
	}

	return rate, nil
}

type RateLimit struct {
	Resources Resources `json:"resources"`
	Rate      Rate      `json:"rate"`
}

type Resources struct {
	Search              Rate `json:"search"`
	Core                Rate `json:"core"`
	GraphQL             Rate `json:"graphql"`
	IntegrationManifest Rate `json:"integration_manifest"`
	CodeScanningUpload  Rate `json:"code_scanning_upload"`
}

type Rate struct {
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"`
	Used      int `json:"used"`
}
