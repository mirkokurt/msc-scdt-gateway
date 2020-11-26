package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func readParamenters(b []string) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	requestUrl := "https://" + *serverAddr + *parametersUrl
	req, err := http.NewRequest("GET", requestUrl, nil)
	req.Header.Add("Content-Type", "application/json")

	var Token = "Bearer " + BearerToken
	//fmt.Printf("Using bearer token: %s \n", Token)

	req.Header.Add("Authorization", Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error during the request of parameters, %v", err)
	} else {
		fmt.Printf("Resp is, %v", resp)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading the body of the response: ", err)
		}
		var payload ParameterPayload
		json.Unmarshal(body, &payload)
		b[0] = hexToString(byte(160)) //0xA0
		b[1] = hexToString(byte(payload.DISTANCE_THR_PARAM))
		b[2] = hexToString(byte(payload.DURATION_THR_PARAM))
		b[3] = hexToString(byte(payload.TX_RATE_PARAM))
		b[4] = hexToString(byte(payload.GRACE_PERIOD_PARAM))
		b[5] = hexToString(byte(payload.SCAN_WINDOW_PARAM))
		b[6] = hexToString(byte(payload.SLEEP_WINDOW_PARAM))
		b[7] = hexToString(byte(payload.PACKET_COMPUTE_DIST_PARAM))
		b[8] = hexToString(byte(payload.ALERTING_DURATION_PARAM))
		b[9] = hexToString(byte(payload.GW_SIGNAL_TIMEOUT_PARAM))
		b[10] = hexToString(byte(payload.GW_PACKET_COMPUTE_DIST_PARAM))
		b[11] = hexToString(byte(payload.GW_AVG_RSSI_PARAM))
		b[12] = hexToString(byte(payload.BLE_TX_POWER_PARAM))
		b[13] = hexToString(byte(payload.QUUPPA_PACKET_COMP_DIST_PARAM))
		b[14] = hexToString(byte(payload.QUUPPA_AVG_RSSI_PARAM))
		b[15] = hexToString(byte(payload.QUUPPA_FORCE_EXIT_PERIOD_PARAM))
		b[16] = hexToString(byte(payload.QUUPPA_TIMEOUT_PARAM & 255))
		b[17] = hexToString(byte(payload.QUUPPA_TIMEOUT_PARAM >> 8))
		b[18] = hexToString(byte(payload.NO_MOVE_ACTIONS_TIMEOUT_PARAM & 255))
		b[19] = hexToString(byte(payload.NO_MOVE_ACTIONS_TIMEOUT_PARAM >> 8))

	}
}

func hexToString(b byte) string {
	if b < 16 {
		return fmt.Sprintf("0%X", b)
	}
	return fmt.Sprintf("%X", b)
}
