package utils

import (
	"encoding/xml"
	"github.com/vmware/govmomi"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type Health struct {
	Status string `xml:"status"`
}

func CheckHealthEndpoint(client *govmomi.Client, healthCheckInterval int) {
	// Async function to fix API timeouts
	var url = strings.Split(client.URL().String(), "/sdk")[0] + "/vapiendpoint/health"
	for {
		response, _ := client.Get(url)
		time.Sleep(time.Duration(healthCheckInterval) * time.Second)
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		var health Health
		err = xml.Unmarshal(data, &health)
		if err != nil {
			log.Fatal(err)
		}
		if health.Status != "GREEN" {
			log.Fatal("API health status is bad")
		}
	}
}
