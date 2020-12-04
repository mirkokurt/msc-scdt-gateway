package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// The function send the contact to a web hook
func sendContactToWebHook(contact StoredContact) {

	var message webContactHookMessage

	message.Event = contact

	jsonContact, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}
	//fmt.Printf("Value: %s", jsonContact)
	
	WebHookURL := "https://" + *serverAddr + WebHookEndpoint

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

// The function send the contact to a web hook
func sendStateToWebHook(state *TagState) {

	var message webStateHookMessage

	message.Event = *state

	jsonContact, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}
	//fmt.Printf("Value: %s", jsonContact)
	
	WebHookURL := "https://" + *serverAddr + WebHookEndpoint

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", WebHookURL, bytes.NewBuffer(jsonContact))
	req.Header.Add("Content-Type", "application/json")
	if APIKey != "" && APIValue != "" {
		req.Header.Add(APIKey, APIValue)
	}

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending the contact to the web hook, %v", err)
		fmt.Printf("Writing in the file\n")
		fileMuX.Lock()
		f, err := os.OpenFile("logfile", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		check(err)
		defer f.Close()
		n3, err := f.WriteString(fmt.Sprintf("%s\n", jsonContact))
		if err != nil {
			fmt.Printf("Error writing in the file %s\n", err)
		}
		fmt.Printf("wrote %d bytes\n", n3)

		f.Sync()
		fileMuX.Unlock()
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
