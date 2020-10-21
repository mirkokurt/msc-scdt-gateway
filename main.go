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
)

var (
	device = flag.String("device", "default", "implementation of ble")
	name   = flag.String("name", "LED", "name of remote peripheral")
	uuid   = flag.String("uuid", "19b10000e8f2537e4f6cd104768a1214", "uiid to search for")
	du     = flag.Duration("du", 30*time.Second, "scanning duration")
	dup    = flag.Bool("dup", true, "allow duplicate reported")
	sub    = flag.Duration("sub", 10*time.Second, "subscribe to notification and indication for a specified period")
	sd     = flag.Duration("sd", 10*time.Second, "scanning duration, 0 for indefinitely")
)

var connectMuX sync.Mutex

func main() {
	flag.Parse()

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
 		b:= []byte{1, 2, 3, 4, 5}
		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *du))
		chkErr(ble.AdvertiseIBeaconData(ctx, b))
		close(stopAdvertise)
	}()


	for j := 0; j < 5; j++ {
	  fmt.Println("Ciclo numero %n", j)
	  go peripheralConnect(filter)
    	}
	fmt.Printf("dopo for\n")
    	<-stopAdvertise
	
}


func advHandler(a ble.Advertisement) {

	if a.LocalName() == "SyncCont" {
		fmt.Printf(" Name: %s", a.LocalName())
	}
	fmt.Printf("\n")
}


func explore(cln ble.Client, p *ble.Profile) error {
	for _, s := range p.Services {
		fmt.Printf("    Service: %s %s, Handle (0x%02X)\n", s.UUID, ble.Name(s.UUID), s.Handle)

		for _, c := range s.Characteristics {
			fmt.Printf("      Characteristic: %s %s, Property: 0x%02X (%s), Handle(0x%02X), VHandle(0x%02X)\n",
				c.UUID, ble.Name(c.UUID), c.Property, propString(c.Property), c.Handle, c.ValueHandle)
			if (c.Property & ble.CharRead) != 0 {
				//fmt.Printf("Time is: %n\n", time.Now().UnixNano())
				b, err := cln.ReadCharacteristic(c)
				if err != nil {
					fmt.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				//fmt.Printf("Time is: %n\n", time.Now().UnixNano())
				fmt.Printf("        Value         %x | %q\n", b, b)
			}

			if c.UUID.Equal(ble.MustParse("19b10001e8f2537e4f6cd104768a1214")) {

				for j := 0; j < 6; j++ {		
					v := j%2
					b := []byte{byte(v)}	
					fmt.Printf("Scrivo valore %n\n", v)			
					cln.WriteCharacteristic(c, b, false)
					time.Sleep(1*time.Second)
			    	}

			}

			for _, d := range c.Descriptors {
				fmt.Printf("        Descriptor: %s %s, Handle(0x%02x)\n", d.UUID, ble.Name(d.UUID), d.Handle)
				b, err := cln.ReadDescriptor(d)
				if err != nil {
					fmt.Printf("Failed to read descriptor: %s\n", err)
					continue
				}
				fmt.Printf("        Value         %x | %q\n", b, b)
			}

			if *sub != 0 {
				// Don't bother to subscribe the Service Changed characteristics.
				if c.UUID.Equal(ble.ServiceChangedUUID) {
					continue
				}

				// Don't touch the Apple-specific Service/Characteristic.
				// Service: D0611E78BBB44591A5F8487910AE4366
				// Characteristic: 8667556C9A374C9184ED54EE27D90049, Property: 0x18 (WN),
				//   Descriptor: 2902, Client Characteristic Configuration
				//   Value         0000 | "\x00\x00"
				if c.UUID.Equal(ble.MustParse("8667556C9A374C9184ED54EE27D90049")) {
					continue
				}

				if (c.Property & ble.CharNotify) != 0 {
					fmt.Printf("\n-- Subscribe to notification for %s --\n", *sub)
					h := func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) }
					if err := cln.Subscribe(c, false, h); err != nil {
						log.Fatalf("subscribe failed: %s", err)
					}
					time.Sleep(*sub)
					if err := cln.Unsubscribe(c, false); err != nil {
						log.Fatalf("unsubscribe failed: %s", err)
					}
					fmt.Printf("-- Unsubscribe to notification --\n")
				}
				if (c.Property & ble.CharIndicate) != 0 {
					fmt.Printf("\n-- Subscribe to indication of %s --\n", *sub)
					h := func(req []byte) { fmt.Printf("Indicated: %q [ % X ]\n", string(req), req) }
					if err := cln.Subscribe(c, true, h); err != nil {
						log.Fatalf("subscribe failed: %s", err)
					}
					time.Sleep(*sub)
					if err := cln.Unsubscribe(c, true); err != nil {
						log.Fatalf("unsubscribe failed: %s", err)
					}
					fmt.Printf("-- Unsubscribe to indication --\n")
				}
			}
		}
		fmt.Printf("\n")
	}
	return nil
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
	
	fmt.Printf("Creo context\n")
	
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), 3600*time.Second))
	
	connectMuX.Lock()
	fmt.Printf("prima di connect\n")
	cln, err := ble.Connect(ctx, filter)
	if err != nil {
		fmt.Printf("can't connect : %s \n", err)
		return
	}
	connectMuX.Unlock()
	fmt.Printf("dopo connect\n")

	// Make sure we had the chance to print out the message.
	done := make(chan struct{})
	// Normally, the connection is disconnected by us after our exploration.
	// However, it can be asynchronously disconnected by the remote peripheral.
	// So we wait(detect) the disconnection in the go routine.
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
	explore(cln, p)

	// Disconnect the connection. (On OS X, this might take a while.)
	//fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
	//cln.CancelConnection()
	fmt.Printf("fine func\n")

	<-done
}
