package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	//"log"
	"net/http"
)

// The function send the contact to a web hook
func sendWebHook(contact StoredContact) {

	var message webHookMessage

	message.Event = contact

	jsonContact, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}
	//fmt.Printf("Value: %s", jsonContact)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", WebHookURL, bytes.NewBuffer(jsonContact))
	req.Header.Add("Content-Type", "application/json")
	if APIKey != "" && APIValue != "" {
		req.Header.Add(APIKey, APIValue)
	}

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending the contact to the web hook, %v", err)
	} 
}
