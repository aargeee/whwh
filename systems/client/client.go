package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/r3labs/sse/v2"
)

func ClientConnect(serverUrl string, hookUrl string) (*sse.Client, string, error) {

	res, err := http.Get(serverUrl + "/createstream")
	if err != nil {
		return nil, "", err
	}
	sid, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, "", err
	}
	cli := sse.NewClient(serverUrl + "/events")

	return cli, string(sid), nil
}

func ClientSubscribe(client *sse.Client, sid string, hookUrl string, deferfunc ...func()) (unsubscribe func(), err error) {

	cxt, cancel := context.WithCancel(context.Background())
	go client.SubscribeWithContext(cxt, sid, createSubscribeFunc(hookUrl, deferfunc...))
	return cancel, nil
}

func createSubscribeFunc(hookUrl string, deferfunc ...func()) func(msg *sse.Event) {
	return func(msg *sse.Event) {
		defer performDefers(deferfunc...)
		req, err := parseIncomingRequest(msg.Data)
		if err != nil {
			println("Err in reading request, ", err.Error())
			return
		}
		req, err = sanitizeIncomingRequest(req, hookUrl)
		if err != nil {
			println("Err in sanitizing request, ", err.Error())
			return
		}

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			println("Err in reading request, ", err.Error())
			return
		}
	}
}

func parseIncomingRequest(data []byte) (*http.Request, error) {
	var dec []byte
	json.Unmarshal(data, &dec)
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(dec)))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func performDefers(deferfunc ...func()) {
	for _, f := range deferfunc {
		f()
	}
}

func sanitizeIncomingRequest(req *http.Request, hookUrl string) (nreq *http.Request, err error) {
	req.URL, err = url.Parse(hookUrl)
	req.Host = req.URL.Host
	if err != nil {
		return nil, err
	}
	req.RequestURI = ""
	return req, nil
}
