package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"
	//"strings"
	"sync"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/pkg/errors"
	//"github.com/go-ble/ble/linux/adv" 
)

var (
	device = flag.String("device", "default", "implementation of ble")
	name   = flag.String("name", "LED", "name of remote peripheral")
	uuid   = flag.String("uuid", "19b10000e8f2537e4f6cd104768a1214", "uiid to search for")
	du     = flag.Duration("du", 30*time.Second, "scanning duration")
	dup    = flag.Bool("dup", true, "allow duplicate reported")
	sub    = flag.Duration("sub", 10*time.Second, "subscribe to notification and indication for a specified period")
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
		if s.UUID.Equal(ble.MustParse("19b10000e8f2537e4f6cd104768a1214")) {
			for _, c := range s.Characteristics {
				if c.UUID.Equal(ble.MustParse("19b10001e8f2537e4f6cd104768a1214")) {
					if *sub != 0 {
						if (c.Property & ble.CharIndicate) != 0 {
							fmt.Printf("\n-- Subscribe to indication of %s --\n", *sub)
							id1 := cln.Addr().String()
							h := func(req []byte ) { 
								fmt.Printf("Indicated: %q [ % X ]\n", string(req), req) 
								fmt.Printf("Address is: %s\n", id1) 
								formatContact(req)															
							}
							if err := cln.Subscribe(c, true, h); err != nil {
								log.Fatalf("subscribe failed: %s", err)
							}
							
							//Test for LED
							for j := 0; j < 6; j++ {		
								v := j%2
								b := []byte{byte(v)}	
								fmt.Printf("Scrivo valore %n\n", v)			
								cln.WriteCharacteristic(c, b, false)
								time.Sleep(1*time.Second)
						    	}

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

func formatContact(req []byte) {
	c := StoredContact{
		ID1:  "iaisjfoaisja",
		ID2:  "ciidajds",
		TS:   12345,
		Dur:  1,
		Room: "ciao",
		Dist: 1,
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
		cln, err := ble.Connect(ctx, filter)
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
