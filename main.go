package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/muka/go-bluetooth/hw"
	log "github.com/sirupsen/logrus"
)

var (
	argAdapterID           = flag.String("adapter_id", "hci0", "ID of the bluetooth adapter")
	name                   = flag.String("name", "LED", "name of remote peripheral")
	serviceUuid            = flag.String("sUuid", "6e0e5437-0c82-4a6c-8c6b-503fad255e03", "uiid to search for")
	characteristicUuid     = flag.String("cUuid", "87c5a1c3-ebe6-426f-8a7d-bdcb710e10fb", "uiid to search for")
	du                     = flag.Duration("du", 60*time.Second, "scanning duration")
	sub                    = flag.Duration("sub", 60*time.Second, "subscribe to notification and indication for a specified period")
	serverAddr             = flag.String("server_addr", "192.168.0.153", "Address of the server with the data collector and other features")
	argWebHook             = flag.String("send_web_hook", ":8088/services/collector", "Send contacts to a web hook")
	parametersUrl          = flag.String("param_url", ":8089/servicesNS/nobody/search/storage/collections/data/kvcollcontactstracing/TAG_PARAMETER", "Url used to recover parameters value")
	argWebHookAPIKey       = flag.String("web_hook_api_key", "Authorization", "Set the key for API authorization")
	argWebHookAPIValue     = flag.String("web_hook_api_value", "Splunk 9fd18e88-3d02-489a-8d88-1d6aac0f6c3e", "Set the calue for API authorization")
	argMaxConnections      = flag.Int("max_connections", 5, "Max number of parallel connections to tags")
	argBearerToken         = flag.String("bearer_token", "", "Token to be used in the request for parameters")
	argGatewayMode         = flag.String("gateway_mode", "internal", "Gateway operational mode (internal/external)")
	argMaxParallelRoutines = flag.Int("max_parallel_routines", 8, "Max number of routines in synchronizing routines pool")
)

var connectMuX sync.Mutex
var fileMuX sync.Mutex

var b []string
var globalDataReceived bool
var httpClient = http.Client{Timeout: 5 * time.Second}

var end = make(chan struct{})

func main() {

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Setup
	flag.Parse()

	WebHookEndpoint = *argWebHook
	APIKey = *argWebHookAPIKey
	APIValue = *argWebHookAPIValue
	MaxConnections = *argMaxConnections
	BearerToken = *argBearerToken
	GatewayMode = *argGatewayMode
	MaxParallelRoutines = *argMaxParallelRoutines
	AdapterID = *argAdapterID

	log.SetLevel(log.TraceLevel)

	btmgmt := hw.NewBtMgmt(AdapterID)

	// set LE mode
	btmgmt.SetPowered(false)
	btmgmt.SetLe(true)
	btmgmt.SetConnectable(false)
	btmgmt.SetPowered(true)

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Set Authentication agent
	// do not reuse agent0 from service
	agent.NextAgentPath()

	//Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}

	ag := agent.NewSimpleAgent()
	err = agent.ExposeAgent(conn, ag, agent.CapKeyboardOnly, true)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Init parameters to be sent to Tags
	b = []string{"00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00"}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Create the routine that send contact and state to Splunk
	SplunkChannel = make(chan SplunkEvent, 5000)
	go storeEvents()

	// Open (or create) persistent storage queue
	OpenQueue()

	// Load existing contacts
	go UploadContactsFromQueue()

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Start advertising
	go advertisingRoutine()

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Discover devices
	log.Infof("Discovering on %s", AdapterID)

	a, err := adapter.NewAdapter1FromAdapterID(AdapterID)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}
	log.Infof("Adapter created")

	// Removes devices from the cache
	flushDevices(a)
	//go cleanDeviceCacheRoutine(a)

	filter := adapter.NewDiscoveryFilter()

	// Search for a specific service
	filter.AddUUIDs("6e0e5437-0c82-4a6c-8c6b-503fad255e03")
	filter.DuplicateData = false

	globalDataReceived = false

	for {

		log.Trace(">>>>>>>>>>>>>>>>>>>>>>>>>>>> Start discover\n")
		discovery, cancel, err := api.Discover(a, &filter)
		if err != nil {
			log.Infof("Error is %v\n", err)
			return
		}
		//defer cancel()

		go restartDiscoverRoutine(cancel, discovery)

		devChan := make(chan *device.Device1)

		//start a pool of synchronize routines
		for i := 0; i < MaxParallelRoutines; i++ {
			go syncronize(devChan, a)
		}

		for ev := range discovery {

			dev, err := device.NewDevice1(ev.Path)
			if err != nil {
				//log.Infof("Scan error is %v\n", err)
				continue
			}
			if dev == nil || dev.Properties == nil {
				continue
			}

			p := dev.Properties

			n := p.Alias
			if p.Name != "" {
				n = p.Name
			}
			log.Infof("Discovered (%s) %s", n, p.Address)

			//Change the passkey using MAC address of the peripheral
			//Eg: 45:E3:7A:03:55:EF -----> 4 4 7 0 5 4
			//passkey := computePassKey(p.Address)
			ag.SetPassKey(123456)
			//ag.SetPassKey(passkey)

			err = connect(dev, ag, AdapterID)
			if err != nil {
				log.Infof("Error in connect, %v\n", err)

				// Remove this device from the cache for reconnection
				log.Trace("Removing from cache ", dev.Path())
				a.RemoveDevice(dev.Path())

				continue
			}

			devChan <- dev
		}
	}
}

func restartDiscoverRoutine(cancel func(), discovery chan *adapter.DeviceDiscovered) {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		log.Trace(">>>>>>>>>>>>>>>>>>>>>>>>>>>> Check if data has been received globally\n")
		if globalDataReceived == false {
			log.Trace(">>>>>>>>>>>>>>>>>>>>>>>>>>>> No data received restarting discovery\n")
			// Stop discover
			cancel()
			break
		}
		globalDataReceived = false
	}
}

func flushDevices(a *adapter.Adapter1) {
	err := a.FlushDevices()
	if err != nil {
		//log.Infof("Error is %v\n", err)
		//return
	}
	log.Infof("Flush device")
}

func syncronize(devChan chan *device.Device1, a *adapter.Adapter1) {

	for {

		dev := <-devChan

		p := dev.Properties

		charact, err := dev.GetCharByUUID("87c5a1c3-ebe6-426f-8a7d-bdcb710e10fb")
		if err != nil {
			log.Infof("Error is %v\n", err)
			return
		}

		err = charact.StartNotify()
		if err != nil {
			log.Infof("Error is %v\n", err)
			return
		}

		log.Infof("Subscribe to characteristic")
		watchProps, err := charact.WatchProperties()
		if err != nil {
			log.Infof("Error is %v\n", err)
			return
		}

		dataReceived := false

		go func() {

			log.Infof("Device address is %s", p.Address)
			id1 := p.Address
			prec := time.Now().UnixNano()
			var ts TagState
			ts.TagID = p.Address

			for propUpdate := range watchProps {

				if propUpdate.Name == "Value" {
					log.Debugf("--> updated %s=%v", propUpdate.Name, propUpdate.Value)

					// Signal that a data has been received
					dataReceived = true
					globalDataReceived = true

					// Calculate and print the passed time
					diff := time.Now().UnixNano() - prec
					log.Trace("Diff is: ", diff)
					prec = time.Now().UnixNano()

					// Format and send the contact to Splunk
					processUpdate(id1, propUpdate.Value.([]byte), &ts)
				}
			}
			SplunkChannel <- ts

			log.Trace("Listener routine stopped")
		}()

		// Check every 5 seconds if at lest one data record has been receive, if not disconnect
		for {
			time.Sleep(5 * time.Second)
			if dataReceived == false {
				break
			}
			dataReceived = false
		}

		log.Trace("Disconnecting from ", p.Address)
		dev.Disconnect()

		// Remove this device from the cache for reconnection
		log.Trace("Removing from cache ", dev.Path())
		a.RemoveDevice(dev.Path())

		log.Trace("Stop the change listener routine")
		close(watchProps)
	}
}

func connect(dev *device.Device1, ag *agent.SimpleAgent, adapterID string) error {

	props, err := dev.GetProperties()
	if err != nil {
		return fmt.Errorf("Failed to load props: %s", err)
	}

	log.Infof("Found device name=%s addr=%s rssi=%d", props.Name, props.Address, props.RSSI)

	if props.Connected {
		log.Info("Device is connected")
		return nil
	}

	if !props.Paired || !props.Trusted {
		log.Info("Pairing device")

		pairTime := time.Now().UnixNano()

		err := dev.Pair()
		if err != nil {

			return fmt.Errorf("Pair failed: %s", err)

		}

		pairTime = time.Now().UnixNano() - pairTime
		log.Trace("Pair time is: ", pairTime)

		log.Info("Pair succeed, connecting...")
		//agent.SetTrusted(adapterID, dev.Path())
	}

	if !props.Connected {
		log.Info("Connecting device")
		err = dev.Connect()
		if err != nil {
			if !strings.Contains(err.Error(), "Connection refused") {
				return fmt.Errorf("Connect failed: %s", err)
			}
		}
	}

	return nil
}

func advertisingRoutine() {

	// Update parameter from Splunk
	go updateParameters()

	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		advertise()
	}
}

func cleanDeviceCacheRoutine(a *adapter.Adapter1) {

	ticker := time.NewTicker(300 * time.Second)
	for range ticker.C {
		log.Trace(">>>>>>>>>>>>>>>>>>>>>>>>>>>> Force cleaning cache \n")
		flushDevices(a)
	}
}

func updateParameters() {
	// Update parameters for the first time
	readParamenters(b)

	// Update parameters periodically
	ticker := time.NewTicker(300 * time.Second)
	for range ticker.C {
		readParamenters(b)
	}
}

func processUpdate(id1 string, b []byte, ts *TagState) {

	id2_check := binary.LittleEndian.Uint32(b[0:4])

	// if id2 = 0, this is a state message
	// payload example { 0, 0, 0, 0, FWVersion_release, FWVersion_minor/build, syncTS, syncTS, syncTS, syncTS, totalContact, totalContact, batteryLevel, batteryLevel }
	if id2_check == 0 {
		fmt.Printf("Array is % X:", b)

		fwVer := fmt.Sprint(b[4]) + "." + fmt.Sprint((b[5]&240)>>4) + "." + fmt.Sprint(b[5]&15)

		syncTS := int64(binary.LittleEndian.Uint32(b[6:10]))
		totalContact := int16(binary.LittleEndian.Uint16(b[10:12]))
		batteryLevel := int16(binary.LittleEndian.Uint16(b[12:14]))
		opMode := uint8((b[14] & 192) >> 6)
		opModeString := ""
		switch opMode {
		case 1:
			opModeString = "Client"
		case 2:
			opModeString = "Worker"
		default:
			opModeString = ""
		}
		paramVersion := int8(b[14] & 63)

		ts.LastSeen = nowTimestamp()
		ts.SyncTime = nowTimestamp() - syncTS
		ts.TotalContact = totalContact
		ts.SyncContact = 0
		ts.BatteryLevel = batteryLevel
		ts.OpMode = opModeString
		ts.ParamVersion = paramVersion
		ts.FWVersion = fwVer

	} else {
		// otherwise it is a contact message
		// payload example { mac, mac, mac, mac, mac, mac, TS, TS, TS, TS, dur dur, avgRSS, zone, zone} : {128, 1, 255, 3, 6, 10, 152, 58, 0, 0, 1, 0, 218, 255, 191}
		id2_string := hex.EncodeToString(b[0:6])
		id2 := id2_string[10:12] + ":" + id2_string[8:10] + ":" + id2_string[6:8] + ":" + id2_string[4:6] + ":" + id2_string[2:4] + ":" + id2_string[0:2]
		startTs := int64(binary.LittleEndian.Uint32(b[6:10]))
		duration := int16(binary.LittleEndian.Uint16(b[10:12]))
		avgRSSI := int8(b[12])

		zoneID := binary.LittleEndian.Uint16(b[13:15])
		room := ""
		// ignore if 0xBFFF
		if zoneID != 49151 {
			room = "Zone" + fmt.Sprint(zoneID)
		}

		adjustedTs := nowTimestamp()
		// If the sync time is found and it is different from 0, compute the adjusted time otherwise use time.now()
		if ts.SyncTime != 0 {
			adjustedTs = startTs + ts.SyncTime
			ts.SyncContact += 1
		} else {
			fmt.Println("Error: no last sync time found for the tag : ", id1)
		}

		c := StoredContact{
			ID1:     id1,
			ID2:     id2,
			TS:      adjustedTs,
			Dur:     duration,
			Room:    room,
			AvgRSSI: avgRSSI,
		}
		// Put the contact into the splunk channel for processing storage
		SplunkChannel <- c
	}
}

func nowTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func storeEvents() {
	for {
		e := <-SplunkChannel

		if e == nil {
			return
		}

		fmt.Printf("Sending event %+v \n", e)
		e.store()
	}
}

func advertise() {

	_, err := exec.Command("sudo", "hcitool", "-i", AdapterID, "cmd", "0x08", "0x0008", "1B", "1A", "ff", "a3", "09", b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19], b[20], b[21], b[22]).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}

	_, err = exec.Command("sudo", "hcitool", "-i", AdapterID, "cmd", "0x08", "0x0006", "90", "00", "90", "00", "06", "00", "00", "00", "00", "00", "00", "00", "00", "07", "00").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}

	_, err = exec.Command("sudo", "hcitool", "-i", AdapterID, "cmd", "0x08", "0x000a", "01").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}

}

func computePassKey(ar string) uint32 {

	log.Trace(">>>>>>>>>>>>>>>>>>>>>>>>>>>> Address is ", ar)

	first, _ := strconv.ParseInt(string(ar[0]), 16, 8)
	second, _ := strconv.ParseInt(string(ar[3]), 16, 8)
	third, _ := strconv.ParseInt(string(ar[6]), 16, 8)
	fourth, _ := strconv.ParseInt(string(ar[9]), 16, 8)
	fifth, _ := strconv.ParseInt(string(ar[12]), 16, 8)
	sixth, _ := strconv.ParseInt(string(ar[15]), 16, 8)

	passkey := uint32(first%10*100000 + second%10*10000 + third%10*1000 + fourth%10*100 + fifth%10*10 + sixth%10)

	log.Trace(">>>>>>>>>>>>>>>>>>>>>>>>>>>> Passkey is ", passkey)

	return passkey
}
