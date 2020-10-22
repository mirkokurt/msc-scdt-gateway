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

// WebHookURL - URL of the specific Web Hook
var WebHookURL string

// WebHookAuthorization - URL of the specific Web Hook
var WebHookAuthorization string

// APIKey - Set the key to be used in the API key authorization
var APIKey string

// APIValue - Set the value to be used in the API key authorization
var APIValue string

// splunkChannel - Channel to be shared between routines in order to store contacts
var splunkChannel chan StoredContact
