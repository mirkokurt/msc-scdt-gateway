package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"
	"strconv"
	"os/exec"
    "runtime"

	//"strings"
	"encoding/binary"
	"encoding/hex"
	"sync"

	"github.com/godbus/dbus"
	"github.com/muka/go-bluetooth/hw"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/muka/go-bluetooth/bluez/profile/advertising"
)

var (
	device = flag.String("device", "default", "implementation of ble")
	name   = flag.String("name", "LED", "name of remote peripheral")
	//uuid   = flag.String("uuid", "19b10000e8f2537e4f6cd104768a1214", "uiid to search for")
	serviceUuid        = flag.String("sUuid", "6e0e5437-0c82-4a6c-8c6b-503fad255e03", "uiid to search for")
	characteristicUuid = flag.String("cUuid", "87c5a1c3-ebe6-426f-8a7d-bdcb710e10fb", "uiid to search for")
	du                 = flag.Duration("du", 60*time.Second, "scanning duration")
	sub                = flag.Duration("sub", 60*time.Second, "subscribe to notification and indication for a specified period")
	serverAddr         = flag.String("server_addr", "192.168.0.153", "Address of the server with the data collector and other features")
	argWebHook         = flag.String("send_web_hook", "https://192.168.0.153:8088/services/collector", "Send contacts to a web hook")
	parametersUrl      = flag.String("param_url", ":8089/servicesNS/nobody/search/storage/collections/data/kvcollcontactstracing/PARAMETER", "Url used to recover parameters value")
	argWebHookAPIKey   = flag.String("web_hook_api_key", "Authorization", "Set the key for API authorization")
	argWebHookAPIValue = flag.String("web_hook_api_value", "Splunk 9fd18e88-3d02-489a-8d88-1d6aac0f6c3e", "Set the calue for API authorization")
	argMaxConnections  = flag.Int("max_connections", 5, "Max number of parallel connections to tags")
	argBearerToken     = flag.String("bearer_token", "", "Token to be used in the request for parameters")
)

var connectMuX sync.Mutex
var fileMuX sync.Mutex

var b []string

var end = make(chan struct{})

func main() {

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Setup
	flag.Parse()

	WebHookURL = *argWebHook
	APIKey = *argWebHookAPIKey
	APIValue = *argWebHookAPIValue
	MaxConnections = *argMaxConnections
	BearerToken = *argBearerToken
	
	adapterID := "hci0"
	
	log.SetLevel(log.TraceLevel)

	btmgmt := hw.NewBtMgmt(adapterID)

	// set LE mode
	btmgmt.SetPowered(false)
	btmgmt.SetLe(true)
	btmgmt.SetBredr(false)
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
	ag.SetPassKey(123456)
	//err = agent.ExposeAgent(conn, ag, agent.CapNoInputNoOutput, true)
	err = agent.ExposeAgent(conn, ag, agent.CapKeyboardOnly, true)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Init parameters to be sent to Tags
	//b = ["00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00", "00"]

	// Create file if it not exists
	if !FileExists("logfile") {
		CreateFile("logfile")
	}
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Create the routine that send contact to Splunk
	SplunkChannel = make(chan StoredContact, 5000)
	go storeContacts(SplunkChannel)
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Start advertising
	go advertising()
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Discover devices
	log.Infof("Discovering on %s", adapterID)

	a, err := adapter.NewAdapter1FromAdapterID(adapterID)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}
	log.Infof("Adapter created")
	
	err = a.FlushDevices()
	if err != nil {
		log.Infof("Error is %v\n", err)
		return 
	}
	log.Infof("Flush device")
	
	filter := adapter.NewDiscoveryFilter()
	
	// Search for a specific service
	filter.AddUUIDs("6e0e5437-0c82-4a6c-8c6b-503fad255e03")
	filter.DuplicateData = true

	discovery, cancel, err := api.Discover(a, &filter)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}

	defer cancel()	

	for ev := range discovery {
		go connect(ev)
	}

}

func connect (ev dbus.ObjectPath){

	dev, err := device.NewDevice1(ev.Path)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return 
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
	
	err = connect(dev, ag, adapterID)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}
	
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
	
	go func() {
		id1 := dev.GetAddress()
		prec := time.Now().UnixNano()
		
		for propUpdate := range watchProps {
		
			log.Debugf("--> updated %s=%v", propUpdate.Name, propUpdate.Value)
			
			// Calculate and print the passed time
			diff := time.Now().UnixNano() - prec
			fmt.Print("Diff is: ")
			fmt.Println(diff)
			prec = time.Now().UnixNano()
			
			// Format and send the contact to Splunk
			formatContact(id1, propUpdate.Value.([]byte))
		}
	}()
	
	<-end
}

//Advertisign routine
func advertising() {

	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		advertise()
	}	

	for {
		// Update parameter from Splunk
		readParamenters(b)
	}
}


func formatContact(id1 string, b []byte) {

	id2_check := uint32(binary.LittleEndian.Uint16(b[0:2])) + binary.LittleEndian.Uint32(b[2:6])

	// if id2 = 0, this is a state message
	// payload example { 0, 0, 0, 0, 0, 0, syncTS, syncTS, syncTS, syncTS, totalContact, totalContact, batteryLevel, batteryLevel }
	if id2_check == 0 {
		fmt.Printf("Arra is % X:", b)
		syncTS := int64(binary.LittleEndian.Uint32(b[6:10]))
		totalContact := int16(binary.LittleEndian.Uint16(b[10:12]))
		batteryLevel := int16(binary.LittleEndian.Uint16(b[12:14]))

		ts, found := tagsState.Load(id1)
		if found {
			ts.(*TagState).LastSeen = nowTimestamp()
			ts.(*TagState).SyncTime = nowTimestamp() - syncTS
			ts.(*TagState).TotalContact = totalContact
			ts.(*TagState).SyncContact = 0
			ts.(*TagState).BatteryLevel = batteryLevel
		} else {
			var its TagState
			its.TagID = id1
			its.LastSeen = nowTimestamp()
			its.SyncTime = nowTimestamp() - syncTS
			its.TotalContact = totalContact
			its.SyncContact = 0
			its.BatteryLevel = batteryLevel
			tagsState.Store(id1, &its)
		}

	} else {
		// otherwise it is a contact message
		// payload example { mac, mac, mac, mac, mac, mac, TS, TS, TS, TS, dur dur, avgRSS, zone, zone} : {128, 1, 255, 3, 6, 10, 152, 58, 0, 0, 1, 0, 218, 255, 191}
		id2_string := hex.EncodeToString(b[0:6])
		id2 := id2_string[10:12] + ":" + id2_string[8:10] + ":" + id2_string[6:8] + ":" + id2_string[4:6] + ":" + id2_string[2:4] + ":" + id2_string[0:2]
		startTs := int64(binary.LittleEndian.Uint32(b[6:10]))
		duration := int16(binary.LittleEndian.Uint16(b[10:12]))
		avgRSSI := int8(b[12])
		room := "Zone_" + fmt.Sprint(binary.LittleEndian.Uint16(b[13:15]))

		adjustedTs := nowTimestamp()
		ts, found := tagsState.Load(id1)
		// If the sync time is found and it is different from 0, compute the adjusted time otherwise use time.now()
		if found {
			adjustedTs = (startTs + ts.(*TagState).SyncTime)
			ts.(*TagState).SyncContact += 1
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


func storeState(TagId string) {
	fmt.Printf("Sending state for tag %s \n", TagId)
	ts, found := tagsState.Load(TagId)
	if found {
		fmt.Printf("Sending state %+v \n", ts)
		sendStateToWebHook(ts.(*TagState))
	}
}

func storeContacts(SplunkChannel chan StoredContact) {
	for {
		c := <-SplunkChannel
		fmt.Printf("Sending contact %+v \n", c)
		sendContactToWebHook(c)

		// Not send too fast
		//time.Sleep(100 * time.Millisecond)
	}
}


func advertise() {
	strconv.FormatInt(255, 16)
	
    out, err := exec.Command("hcitool", "-i", "hci0", "cmd", "0x08", "0x0008", "18", "17", "ff", "a3", "09", b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]).Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
	
    out, err = exec.Command("hcitool", "-i", "hci0", "cmd", "0x08", "0x0006", "A0", "00", "B0", "00", "03", "00", "00", "00", "00", "00", "00", "00", "00", "07", "00").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
	
	out, err = exec.Command("hcitool", "-i", "hci0", "cmd", "0x08", "0x000a", "01").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }

}
