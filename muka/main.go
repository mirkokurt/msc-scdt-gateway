package main

import (
	"os"
	"time"
	"fmt"
	"strings"

    "github.com/godbus/dbus"
	"github.com/muka/go-bluetooth/hw"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/muka/go-bluetooth/bluez/profile/advertising"
	//"github.com/muka/go-bluetooth/api/service"
	//"github.com/muka/go-bluetooth/bluez/profile/gatt"
	log "github.com/sirupsen/logrus"
)

var end = make(chan struct{})

func main()  {

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Setup

	adapterID := "hci0"
	
	log.SetLevel(log.TraceLevel)

	btmgmt := hw.NewBtMgmt(adapterID)
	if len(os.Getenv("DOCKER")) > 0 {
		btmgmt.BinPath = "./bin/docker-btmgmt"
	}

	// set LE mode
	btmgmt.SetPowered(false)
	btmgmt.SetLe(true)
	btmgmt.SetBredr(false)
	btmgmt.SetConnectable(false)
	btmgmt.SetPowered(true)
	
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Set Authentication
	// do not reuse agent0 from service
	agent.NextAgentPath()
	
	//Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}

	ag := agent.NewSimpleAgent()
	ag.SetPassKey(123456)
	err = agent.ExposeAgent(conn, ag, agent.CapNoInputNoOutput, true)
	if err != nil {
		return
	}
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Discover devices
	log.Infof("Discovering on %s", adapterID)

	a, err := adapter.NewAdapter1FromAdapterID(adapterID)
	if err != nil {
		return
	}
	log.Infof("Adapter created")
	
	err = a.FlushDevices()
	if err != nil {
		return 
	}
	log.Infof("Flush device")
	
	filter := adapter.NewDiscoveryFilter()
	//filter.AddUUIDs("0000fd6f-0000-1000-8000-00805f9b34fc")
	filter.AddUUIDs("6e0e5437-0c82-4a6c-8c6b-503fad255e03")
	filter.DuplicateData = true


	discovery, cancel, err := api.Discover(a, &filter)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}
	
	log.Infof("Discovery called")

	defer cancel()

	
	go func() {
		for ev := range discovery {
			dev, err := device.NewDevice1(ev.Path)
			if err != nil {
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
				return
			}
			
			log.Info("Listing exposed services")
			retrieveServices(a, dev)
			
			/*
			watchProps, err := dev.WatchProperties()
			if err != nil {
				return
			}
			log.Infof(">>>>>>>>>>>>>>>>>> Start watch properties !")
			go func() {
				for propUpdate := range watchProps {
					log.Debugf("--> updated %s=%v", propUpdate.Name, propUpdate.Value)
				}
			}()
			*/
		}
	}()
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Start Advertising
	go startAdvertising(adapterID)
	
	<-end
	log.Infof("End of the program ")
	// return
}

func retrieveServices(a *adapter.Adapter1, dev *device.Device1) error {

	log.Info("Listing exposed services")

	list, err := dev.GetAllServicesAndUUID()
	if err != nil {
		return err
	}
	
	log.Info("Result \n", list)

	if len(list) == 0 {
		time.Sleep(time.Second * 2)
		
		return retrieveServices(a, dev)
	}

	for _, servicePath := range list {
		log.Info("%s", servicePath)
	}

	return nil
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

		err := dev.Pair()
		if err != nil {
			return fmt.Errorf("Pair failed: %s", err)
		}

		log.Info("Pair succeed, connecting...")
		agent.SetTrusted(adapterID, dev.Path())
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

func startAdvertising(adapterID string) {

	var prop *advertising.LEAdvertisement1Properties
	prop = new(advertising.LEAdvertisement1Properties)
	prop.Appearance = 512
	prop.Discoverable = false
	prop.DiscoverableTimeout = 0
	prop.Duration = 1
	
	prop.LocalName = "ContactsGateway"

	b := []uint8{1, 2, 3, 4, 5, 6, 7}
	prop.AddManifacturerData(555, b)
	
	prop.Timeout = 300
	prop.Type = "broadcast"
	
	log.Infof("Prop is \n", prop)
	
	//interf, _ := prop.ToMap()
	
	//log.Infof("Interf is %v\n", interf)
	
	cancelAdv, err := api.ExposeAdvertisement(adapterID, prop, 0)
	if err != nil {
		log.Infof("Error is %v\n", err)
		return
	}
	defer cancelAdv()
	<-end
	
	log.Infof("End Advertising")
}

	