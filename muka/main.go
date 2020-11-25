package main

import (
	"time"
	"fmt"
	"strings"
	"os/exec"
    "runtime"

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
			
			retrieveServices(a, dev)
			
			chList, err := dev.GetCharacteristicsList()
			if err != nil {
				log.Infof("Error is %v\n", err)
				return
			}
			
			log.Infof("Charact list is %v \n", n, chList)
			/*
			charact, err := gatt.NewGattCharacteristic1("/org/bluez/hci0/dev_E5_BA_F4_30_D0_CC/service000e")
			if err != nil {
				log.Infof("Error is %v\n", err)
				return
			}
			*/
			
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
		        
			
			watchProps, err := charact.WatchProperties()
			if err != nil {
				log.Infof("Error is %v\n", err)
				return
			}
			
			
			log.Infof(">>>>>>>>>>>>>>>>>> Start watch properties !")
			go func() {
				for propUpdate := range watchProps {
					log.Debugf("--> updated %s=%v", propUpdate.Name, propUpdate.Value)
				}
			}()
			
		}
	}()
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Start Advertising
	//go startAdvertising(adapterID)
	
	sendCommand()
	
	<-end
	log.Infof("End of the program ")
	// return
}

func retrieveServices(a *adapter.Adapter1, dev *device.Device1) error {

	log.Info("Listing exposed services")

	list, err := dev.GetAllServicesAndUUID()
	if err != nil {
		log.Infof("Error is %v\n", err)
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
	prop.Duration = 5
	
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
	
	//sendCommand()
	<-end
	
	log.Infof("End Advertising")
}

func sendCommand() {
	if runtime.GOOS == "windows" {
        fmt.Println("Can't Execute this on a windows machine")
    } else {
        execute()
    }
}

func execute() {

	/*
    out, err := exec.Command("hciconfig", "hci0", "down").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
    output := string(out[:])
    fmt.Println(output)
	*/
	
	/*
    out, err = exec.Command("hciconfig", "hci0", "up").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
    output = string(out[:])
    fmt.Println(output)
	*/
	
    out, err := exec.Command("hcitool", "-i", "hci0", "cmd", "0x08", "0x0008", "18", "17", "ff", "a3", "09", "01", "02", "03", "04", "05", "06", "07", "08", "09", "0A", "0B", "0C", "0D", "0E", "0F", "10", "11", "12", "13", "15").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
    output := string(out[:])
    fmt.Println(output)
	
    out, err = exec.Command("hcitool", "-i", "hci0", "cmd", "0x08", "0x0006", "A0", "00", "B0", "00", "03", "00", "00", "00", "00", "00", "00", "00", "00", "07", "00").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
    output = string(out[:])
    fmt.Println(output)
	
	out, err = exec.Command("hcitool", "-i", "hci0", "cmd", "0x08", "0x000a", "01").Output()
    if err != nil {
        fmt.Printf("%s", err)
    }
    output = string(out[:])
    fmt.Println(output)

}

	