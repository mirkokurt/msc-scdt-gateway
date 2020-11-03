package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	//"strings"
	"encoding/binary"
	"encoding/hex"
	"sync"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/pkg/errors"
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

func main() {
	flag.Parse()

	WebHookURL = *argWebHook
	APIKey = *argWebHookAPIKey
	APIValue = *argWebHookAPIValue
	MaxConnections = *argMaxConnections
	BearerToken = *argBearerToken

	if !FileExists("logfile") {
		CreateFile("logfile")
	}

	//file, err := os.Create("logfile")
	//check(err)
	//defer file.Close()

	//log.SetOutput(f)
	//log.Println("This is a test log entry")

	SplunkChannel = make(chan StoredContact, 5000)

	go storeContacts(SplunkChannel)

	d, err := dev.NewDevice(*device)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	// Default to search device with a service with UUID specified by user
	filter := func(a ble.Advertisement) bool {
		for _, s := range a.Services() {
			if s.Equal(ble.MustParse(*serviceUuid)) {
				return true
			}
		}
		return false
	}

	stopAdvertise := make(chan struct{})
	go advertising(d)

	for j := 0; j < MaxConnections; j++ {
		//fmt.Printf("Routine number: %d \n", j)
		go peripheralConnect(filter)
	}

	<-stopAdvertise

}

//Start advertising
func advertising(d ble.Device) {
	b := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for {
		readParamenters(b)
		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), 60*5*time.Second))
		chkErr(d.AdvertiseMfgData(ctx, 555, b))
	}
}

func exploreAndSubscribe(cln ble.Client, p *ble.Profile) error {
	for _, s := range p.Services {
		fmt.Printf("    Service: %s %s, Handle (0x%02X)\n", s.UUID, ble.Name(s.UUID), s.Handle)
		//if s.UUID.Equal(ble.MustParse("19b10000e8f2537e4f6cd104768a1214")) {
		if s.UUID.Equal(ble.MustParse(*serviceUuid)) {
			for _, c := range s.Characteristics {
				//if c.UUID.Equal(ble.MustParse("19b10001e8f2537e4f6cd104768a1214")) {
				if c.UUID.Equal(ble.MustParse(*characteristicUuid)) {
					if *sub != 0 {
						if (c.Property & ble.CharIndicate) != 0 {
							fmt.Printf("\n-- Subscribe to indication of %s --\n", *sub)
							id1 := cln.Addr().String()
							prec := time.Now().UnixNano()
							h := func(req []byte) {
								fmt.Printf("Address is: %s\n", id1)
								diff := time.Now().UnixNano() - prec
								fmt.Print("Diff is: ")
								fmt.Println(diff)
								prec = time.Now().UnixNano()
								formatContact(id1, req)
							}
							if err := cln.Subscribe(c, true, h); err != nil {
								log.Fatalf("subscribe failed: %s", err)
							}

							<-cln.Disconnected()
						}
					}
				}
			}
		}
	}
	return nil
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

func propString(p ble.Property) string {
	var s string
	for k, v := range map[ble.Property]string{
		ble.CharBroadcast:   "B",
		ble.CharRead:        "R",
		ble.CharWriteNR:     "w",
		ble.CharWrite:       "W",
		ble.CharNotify:      "N",
		ble.CharIndicate:    "I",
		ble.CharSignedWrite: "S",
		ble.CharExtended:    "E",
	} {
		if p&k != 0 {
			s += v
		}
	}
	return s
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}

func peripheralConnect(filter func(ble.Advertisement) bool) {

	for {
		fmt.Printf("Routine started\n")
		ctx := context.Background()

		connectMuX.Lock()
		cln, err := ConnectWithDuplicate(ctx, filter)
		if err != nil {
			fmt.Printf("can't connect : %s \n", err)
			continue
		}
		connectMuX.Unlock()
		StartedConnections++

		// Define a channel to intercept the end fo communication if its closed by the device
		closedConnection := make(chan struct{})
		go func() {
			<-cln.Disconnected()
			tagId := cln.Addr().String()
			fmt.Printf("[ %s ] is disconnected \n", tagId)
			storeState(tagId)
			close(closedConnection)
		}()

		fmt.Printf("Discovering profile...\n")
		p, err := cln.DiscoverProfile(true)
		if err != nil {
			fmt.Printf("can't discover profile: %s \n", err)
			continue
		}

		// Start the exploration.
		exploreAndSubscribe(cln, p)

		<-closedConnection
	}
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

// Connect searches for and connects to a Peripheral which matches specified condition.
func ConnectWithDuplicate(ctx context.Context, f ble.AdvFilter) (ble.Client, error) {
	ctx2, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-ctx.Done():
			//cancel()
		case <-ctx2.Done():
		}
	}()

	ch := make(chan ble.Advertisement)
	fn := func(a ble.Advertisement) {
		cancel()
		ch <- a
	}
	fmt.Printf("Starting scanning \n")
	if err := ble.Scan(ctx2, true, fn, f); err != nil {
		if err != context.Canceled {
			return nil, errors.Wrap(err, "can't scan")
		}
	}
	fmt.Printf("Starting a connection \n")
	cln, err := ble.Dial(ctx, (<-ch).Addr())
	return cln, errors.Wrap(err, "can't dial")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
