package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"
	//"strings"
	"sync"
	"encoding/binary"
	"encoding/hex"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/pkg/errors"
	//"github.com/go-ble/ble/linux/adv" 
)

var (
	device = flag.String("device", "default", "implementation of ble")
	name   = flag.String("name", "LED", "name of remote peripheral")
	//uuid   = flag.String("uuid", "19b10000e8f2537e4f6cd104768a1214", "uiid to search for")
	uuid   = flag.String("uuid", "6e0e5437-0c82-4a6c-8c6b-503fad255e03", "uiid to search for")
	du     = flag.Duration("du", 60*time.Second, "scanning duration")
	dup    = flag.Bool("dup", true, "allow duplicate reported")
	sub    = flag.Duration("sub", 60*time.Second, "subscribe to notification and indication for a specified period")
	sd     = flag.Duration("sd", 10*time.Second, "scanning duration, 0 for indefinitely")
	argWebHook = flag.String("send_web_hook", "https://webhook.site/222fae5c-dab0-4018-92a4-d1bf5aefb3bd", "Send contacts to a web hook")
	argWebHookAPIKey = flag.String("web_hook_api_key", "Authorization", "Set the key for API authorization")
	argWebHookAPIValue = flag.String("web_hook_api_value", "Splunk 9fd18e88-3d02-489a-8d88-1d6aac0f6c3e", "Set the calue for API authorization")
)

var connectMuX sync.Mutex

func main() {
	flag.Parse()
	
	WebHookURL = *argWebHook
	APIKey = *argWebHookAPIKey
	APIValue = *argWebHookAPIValue
	
	splunkChannel = make(chan StoredContact, 5000)
	
	go storeContacts(splunkChannel)

	d, err := dev.NewDevice(*device)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(d)

	// Default to search device with a service with UUID = "19b10000e8f2537e4f6cd104768a1214" (or specified by user).
	filter := func(a ble.Advertisement) bool {
		//return strings.ToUpper(a.LocalName()) == strings.ToUpper(*name)
		for _, s := range a.Services() {
			if s.Equal(ble.MustParse(*uuid)) {
				return true
			}	
		}
		return false
	}

	// Scan for specified durantion, or until interrupted by user.
	//fmt.Printf("Scanning for %s...\n", *du)
	//chkErr(ble.Scan(ctx, *dup, advHandler, nil))

	stopAdvertise := make(chan struct{})
	//Start advertising
	go func() {
 		b:= []byte{1, 2, 3, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5 }
		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *du))
		chkErr(d.AdvertiseMfgData(ctx, 555, b))
		//ad, _ := adv.NewPacket(adv.Flags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly))
		//ad.Append(adv.CompleteName("Contact Gateway"))
		//manufacuturerData := adv.ManufacturerData(555, b)
		//ad.Append(manufacuturerData)
		//d.HCI.SetAdvertisement(ad.Bytes(), nil)
		//d.HCI.Advertise()
		//chkErr(d.Advertise(ctx, ad))
		
		close(stopAdvertise)
	}()


	for j := 0; j < 5; j++ {
	  go peripheralConnect(filter)
    	}


    	<-stopAdvertise
	
}


func advHandler(a ble.Advertisement) {

	if a.LocalName() == "SyncCont" {
		fmt.Printf(" Name: %s", a.LocalName())
	}
	fmt.Printf("\n")
}


func exploreAndSubscribe(cln ble.Client, p *ble.Profile) error {
	for _, s := range p.Services {
		fmt.Printf("    Service: %s %s, Handle (0x%02X)\n", s.UUID, ble.Name(s.UUID), s.Handle)
		//if s.UUID.Equal(ble.MustParse("19b10000e8f2537e4f6cd104768a1214")) {
		if s.UUID.Equal(ble.MustParse("6e0e5437-0c82-4a6c-8c6b-503fad255e03")) {
			for _, c := range s.Characteristics {
				//if c.UUID.Equal(ble.MustParse("19b10001e8f2537e4f6cd104768a1214")) {
				if c.UUID.Equal(ble.MustParse("87c5a1c3-ebe6-426f-8a7d-bdcb710e10fb")) {
					if *sub != 0 {
						if (c.Property & ble.CharIndicate) != 0 {
							fmt.Printf("\n-- Subscribe to indication of %s --\n", *sub)
							id1 := cln.Addr().String()
							h := func(req []byte ) { 
								fmt.Print("Time is: ")
								fmt.Println(time.Now().UnixNano())
								fmt.Printf("Indicated: %q [ % X ]\n", string(req), req) 
								fmt.Printf("Address is: %s\n", id1) 
								formatContact(id1, req)															
							}
							if err := cln.Subscribe(c, true, h); err != nil {
								log.Fatalf("subscribe failed: %s", err)
							}
							/*
							//Test for LED
							for j := 0; j < 6; j++ {		
								v := j%2
								b := []byte{byte(v)}	
								fmt.Printf("Scrivo valore %n\n", v)			
								cln.WriteCharacteristic(c, b, false)
								time.Sleep(1*time.Second)
						    	}
							*/
							time.Sleep(*sub)
							if err := cln.Unsubscribe(c, true); err != nil {
								log.Fatalf("unsubscribe failed: %s", err)
							}
							fmt.Printf("-- Unsubscribe to indication --\n")
						}
					}
				}
			}
		}
	}
	return nil
}

func formatContact(id1 string, b []byte) {



	//b := []byte{128, 1, 255, 3, 6, 10, 152, 58, 0, 0, 1, 0, 218, 255, 191}

	// payload example { mac, mac, mac, mac, mac, mac, TS, TS, TS, TS, dur dur, avgRSS, zone, zone }

	id2_string := hex.EncodeToString(b[0:6])
	fmt.Println("id_string is: %s \n", id2_string)
	id2 := id2_string[10:12] + ":" + id2_string[8:10] + ":" + id2_string[6:8] + ":" + id2_string[4:6] + ":" + id2_string[2:4] + ":" + id2_string[0:2]
	startTs := int64(binary.LittleEndian.Uint32(b[6:10]))
	duration := int16(binary.LittleEndian.Uint16(b[10:12]))
	avgRSSI := int8(b[12])
	room := "Zone_" + string(binary.LittleEndian.Uint16(b[13:15]))

	c := StoredContact{
		ID1:  id1,
		ID2:  id2,
		TS:   startTs,
		Dur:  duration,
		Room: room,
		AvgRSSI: avgRSSI,
	}
	// Put the contact into the splunk channel for processing storage
	splunkChannel <- c

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
	
	
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), 60*time.Second))

	for {
	
		connectMuX.Lock()
		fmt.Printf("prima di connect\n")
		cln, err := ConnectWithDuplicate(ctx, filter)
		if err != nil {
			fmt.Printf("can't connect : %s \n", err)
			return
		}
		connectMuX.Unlock()
		fmt.Printf("dopo connect\n")

		// Define a channel to intercept the end fo communication if its closed by the device
		done := make(chan struct{})
		go func() {
			<-cln.Disconnected()
			fmt.Printf("[ %s ] is disconnected \n", cln.Addr())
			close(done)
		}()

		fmt.Printf("Discovering profile...\n")
		p, err := cln.DiscoverProfile(true)
		if err != nil {
			fmt.Printf("can't discover profile: %s \n", err)
			return
		}
		

		// Start the exploration.
		exploreAndSubscribe(cln, p)

		<-done
	}
}



func storeContacts(splunkChannel chan StoredContact) {
	for {
		c := <-splunkChannel
		//fmt.Println(c)

		sendWebHook(c)

		// Not send too fast
		time.Sleep(1 * time.Second)
	}
}

// Connect searches for and connects to a Peripheral which matches specified condition.
func ConnectWithDuplicate(ctx context.Context, f ble.AdvFilter) (ble.Client, error) {
	ctx2, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-ctx2.Done():
		}
	}()

	ch := make(chan ble.Advertisement)
	fn := func(a ble.Advertisement) {
		cancel()
		ch <- a
	}
	if err := ble.Scan(ctx2, true, fn, f); err != nil {
		if err != context.Canceled {
			return nil, errors.Wrap(err, "can't scan")
		}
	}

	cln, err := ble.Dial(ctx, (<-ch).Addr())
	return cln, errors.Wrap(err, "can't dial")
}


