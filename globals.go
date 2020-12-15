package main

// webContactHookMessage json body to be sent to the webhook
type webContactHookMessage struct {
	Event StoredContact `json:"event"`
}

// WebStateHookMessage json body to be sent to the webhook
type webStateHookMessage struct {
	Event TagState `json:"event"`
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

// TagState - Last seen state of a tag
type TagState struct {
	TagID        string `json:"ID"`
	LastSeen     int64  `json:"LastSeenTS,omitempty"`
	SyncTime     int64  `json:"-"`
	BatteryLevel int16  `json:"BatteryLevel,omitempty"`
	TotalContact int16  `json:"ToSync,omitempty"`
	SyncContact  int16  `json:"Synced,omitempty"`
	OpMode       string `json:"TagMode,omitempty"`
	ParamVersion int8   `json:"ParamVersion,omitempty"`
	FWVersion    string `json:"FWVersion,omitempty"`
}

type SplunkEvent interface {
	store()
}

func (e StoredContact) store() {
	sendContactToWebHook(e)
}

func (e TagState) store() {
	sendStateToWebHook(e)
}

type ParameterPayload struct {
	VERSION                          uint8  `json:"VERSION"`
	DISTANCE_THR_PARAM               int8   `json:"DISTANCE_THR_PARAM"`
	DURATION_THR_PARAM               uint8  `json:"DURATION_THR_PARAM"`
	TX_RATE_PARAM                    uint8  `json:"TX_RATE_PARAM"`
	GRACE_PERIOD_PARAM               uint8  `json:"GRACE_PERIOD_PARAM"`
	SCAN_WINDOW_PARAM                uint8  `json:"SCAN_WINDOW_PARAM"`
	SLEEP_WINDOW_PARAM               uint8  `json:"SLEEP_WINDOW_PARAM"`
	PACKET_COMPUTE_DIST_PARAM        uint8  `json:"PACKET_COMPUTE_DIST_PARAM"`
	ALERTING_DURATION_PARAM          uint8  `json:"ALERTING_DURATION_PARAM"`
	GW_SIGNAL_TIMEOUT_PARAM          uint8  `json:"GW_SIGNAL_TIMEOUT_PARAM"`
	GW_PACKET_COMPUTE_DIST_PARAM     uint8  `json:"GW_PACKET_COMPUTE_DIST_PARAM"`
	GW_AVG_RSSI_PARAM                int8   `json:"GW_AVG_RSSI_PARAM"`
	BLE_TX_POWER_PARAM               int8   `json:"BLE_TX_POWER_PARAM"`
	QUUPPA_PACKET_COMP_DIST_PARAM    uint8  `json:"QUUPPA_PACKET_COMP_DIST_PARAM"`
	QUUPPA_AVG_RSSI_PARAM            int8   `json:"QUUPPA_AVG_RSSI_PARAM"`
	QUUPPA_FORCE_EXIT_PERIOD_PARAM   uint8  `json:"QUUPPA_FORCE_EXIT_PERIOD_PARAM"`
	QUUPPA_TIMEOUT_PARAM             uint16 `json:"QUUPPA_TIMEOUT_PARAM"`
	NO_MOVE_ACTIONS_TIMEOUT_PARAM    uint16 `json:"NO_MOVE_ACTIONS_TIMEOUT_PARAM"`
	INTERNAL_TAG_STARTUP_DELAY_PARAM uint8  `json:"INTERNAL_TAG_STARTUP_DELAY_PARAM"`
	EXTERNAL_TAG_STARTUP_DELAY_PARAM uint8  `json:"EXTERNAL_TAG_STARTUP_DELAY_PARAM"`
	ACC_PARAM                        uint8  `json:"ACC_PARAM"`
}

// WebHookEndpoint - endpoint of the specific Web Hook
var WebHookEndpoint string

// WebHookAuthorization - URL of the specific Web Hook
var WebHookAuthorization string

// APIKey - Set the key to be used in the API key authorization
var APIKey string

// APIValue - Set the value to be used in the API key authorization
var APIValue string

// SplunkChannel - Channel to be shared between routines in order to store contacts
var SplunkChannel chan SplunkEvent

// SplunkAddress - Ip address of the Splunk server
var SplunkAddress string

// MaxConnections - Max number of parallel connections to tags
var MaxConnections int

// StartedConnections - Number of started parallel connections to tags
var StartedConnections int

// StartedRoutines - Number of started parallel connection routines
var StartedRoutines int

// BearerToken - Token to be used in the request for parameters
var BearerToken string

// GatewayMode - Gateway operational mode (Internal/External)
var GatewayMode string

// MaxParallelRoutines - Max number of routines in synchronizing routines pool
var MaxParallelRoutines int

// AdapterID - ID of the bluetooth adapter
var AdapterID string
