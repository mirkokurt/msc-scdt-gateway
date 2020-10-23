package main

// WebHookMessage json body to be sent to the webhook
type webHookMessage struct {
	Event StoredContact `json:"event"`
}

// StoredContact - Contact to be stored into Cloud or Database
type StoredContact struct {
	ID1     string  `json:"ID1"`
	ID2     string  `json:"ID2"`
	TS      int64   `json:"TS"`
	Dur     int16   `json:"dur"`
	Room    string  `json:"room,omitempty"`
	Dist    float64 `json:"dist,omitempty"`
	AvgRSSI int8    `json:"avgRSSI,omitempty"`
}

type ParameterPayload struct {
	DISTANCE_THR_PARAM             int8  `json:"DISTANCE_THR_PARAM"`
	DURATION_THR_PARAM             int8  `json:"DURATION_THR_PARAM"`
	TX_RATE_PARAM                  int8  `json:"TX_RATE_PARAM"`
	GRACE_PERIOD_PARAM             int8  `json:"GRACE_PERIOD_PARAM"`
	SCAN_WINDOW_PARAM              int8  `json:"SCAN_WINDOW_PARAM"`
	SLEEP_WINDOW_PARAM             int8  `json:"SLEEP_WINDOW_PARAM"`
	PACKET_COMPUTE_DIST_PARAM      int8  `json:"PACKET_COMPUTE_DIST_PARAM"`
	ALERTING_DURATION_PARAM        int8  `json:"ALERTING_DURATION_PARAM"`
	GW_SIGNAL_TIMEOUT_PARAM        int8  `json:"GW_SIGNAL_TIMEOUT_PARAM"`
	GW_PACKET_COMPUTE_DIST_PARAM   int8  `json:"GW_PACKET_COMPUTE_DIST_PARAM"`
	GW_AVG_RSSI_PARAM              int8  `json:"GW_AVG_RSSI_PARAM"`
	BLE_TX_POWER_PARAM             int8  `json:"BLE_TX_POWER_PARAM"`
	QUUPPA_PACKET_COMP_DIST_PARAM  int8  `json:"QUUPPA_PACKET_COMP_DIST_PARAM"`
	QUUPPA_AVG_RSSI_PARAM          int8  `json:"QUUPPA_AVG_RSSI_PARAM"`
	QUUPPA_FORCE_EXIT_PERIOD_PARAM int8  `json:"QUUPPA_FORCE_EXIT_PERIOD_PARAM"`
	QUUPPA_TIMEOUT_PARAM           int16 `json:"QUUPPA_TIMEOUT_PARAM"`
	NO_MOVE_ACTIONS_TIMEOUT_PARAM  int16 `json:"NO_MOVE_ACTIONS_TIMEOUT_PARAM"`
}

// WebHookURL - URL of the specific Web Hook
var WebHookURL string

// WebHookAuthorization - URL of the specific Web Hook
var WebHookAuthorization string

// APIKey - Set the key to be used in the API key authorization
var APIKey string

// APIValue - Set the value to be used in the API key authorization
var APIValue string

// splunkChannel - Channel to be shared between routines in order to store contacts
var SplunkChannel chan StoredContact

// SplunkAddress - Channel to be shared between routines in order to store contacts
var SplunkAddress string
