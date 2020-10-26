package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var bearerToken = "Bearer xxx"

func readParamenters() (b []byte) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	requestUrl := "https://" + *serverAddr + *parametersUrl

	req, err := http.NewRequest("GET", requestUrl, nil)
	req.Header.Add("Content-Type", "application/json")

	req.Header.Add("Authorization", APIValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending the contact to the web hook, %v", err)
	} else {
		fmt.Printf("Resp is, %v", resp)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading the body of the response: ", err)
			return b
		}
		var payload ParameterPayload
		json.Unmarshal(body, &payload)
		b[0] = byte(160) //0xA0
		b[1] = byte(payload.DISTANCE_THR_PARAM)
		b[2] = byte(payload.DURATION_THR_PARAM)
		b[3] = byte(payload.TX_RATE_PARAM)
		b[4] = byte(payload.GRACE_PERIOD_PARAM)
		b[5] = byte(payload.SCAN_WINDOW_PARAM)
		b[6] = byte(payload.SLEEP_WINDOW_PARAM)
		b[7] = byte(payload.PACKET_COMPUTE_DIST_PARAM)
		b[8] = byte(payload.ALERTING_DURATION_PARAM)
		b[9] = byte(payload.GW_SIGNAL_TIMEOUT_PARAM)
		b[10] = byte(payload.GW_PACKET_COMPUTE_DIST_PARAM)
		b[11] = byte(payload.GW_AVG_RSSI_PARAM)
		b[12] = byte(payload.BLE_TX_POWER_PARAM)
		b[13] = byte(payload.QUUPPA_PACKET_COMP_DIST_PARAM)
		b[14] = byte(payload.QUUPPA_AVG_RSSI_PARAM)
		b[15] = byte(payload.QUUPPA_FORCE_EXIT_PERIOD_PARAM)
		b[16] = byte(payload.QUUPPA_TIMEOUT_PARAM)
		b[18] = byte(payload.NO_MOVE_ACTIONS_TIMEOUT_PARAM)

	}
	return b
}
