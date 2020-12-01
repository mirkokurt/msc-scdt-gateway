package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"os"

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
	btmgmt.SetPowered(true)
	
	
	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>> Set Authentication
	// do not reuse agent0 from service
	agent.NextAgentPath()

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

	//Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}
	
		err := a.FlushDevices()
	if err != nil {
		return nil, err
	}
	
	var filter a.DiscoveryFilter
	filter.UUIDs.add("00001801-0000-1000-8000-00805f9b34fb")
	filter.DuplicateData = true

	discovery, cancel, err := api.Discover(a, &filter)
	if err != nil {
		return nil, err
	}

	defer cancel()

	for ev := range discovery {

		log.Infof("Found device %v\n", ev)
	}
}
	