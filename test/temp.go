	for j := 0; j < 5; j++ {
	  fmt.Println("Ciclo numero %n", j)
	  ctx1 := ble.WithSigHandler(context.WithTimeout(context.Background(), *sd))
	  go func() {
		
		cln, err := ble.Connect(ctx1, filter)
		if err != nil {
			log.Fatalf("can't connect : %s", err)
		}

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
			log.Fatalf("can't discover profile: %s", err)
		}

		// Start the exploration.
		explore(cln, p)

		// Disconnect the connection. (On OS X, this might take a while.)
		fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
		//cln.CancelConnection()


		<-done
	   }()
    	}
    <-stopAdvertise

____________________________________________________________

	ctx1 := ble.WithSigHandler(context.WithTimeout(context.Background(), *sd))
	cln, err := ble.Connect(ctx1, filter)
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

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
		log.Fatalf("can't discover profile: %s", err)
	}

	// Start the exploration.
	explore(cln, p)


	ctx2 := ble.WithSigHandler(context.WithTimeout(context.Background(), *sd))
	cln1, err := ble.Connect(ctx2, filter)
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

	// Make sure we had the chance to print out the message.
	done1 := make(chan struct{})
	// Normally, the connection is disconnected by us after our exploration.
	// However, it can be asynchronously disconnected by the remote peripheral.
	// So we wait(detect) the disconnection in the go routine.
	go func() {
		<-cln1.Disconnected()
		fmt.Printf("[ %s ] is disconnected \n", cln.Addr())
		close(done1)
	}()

	fmt.Printf("Discovering profile...\n")
	p1, err := cln1.DiscoverProfile(true)
	if err != nil {
		log.Fatalf("can't discover profile: %s", err)
	}

	// Start the exploration.
	explore(cln1, p1)

	// Disconnect the connection. (On OS X, this might take a while.)
	//fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
	//cln.CancelConnection()

	// Disconnect the connection. (On OS X, this might take a while.)
	//fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Addr())
	//cln1.CancelConnection()

	<-stopAdvertise
	<-done
	<-done1

_________________________________________________________________________________

		/*
		uuidServiceFilter := []ble.UUID {ble.MustParse("19b10000e8f2537e4f6cd104768a1214")}
		uuidCharacteristicsFilter :=  []ble.UUID {ble.MustParse("19b10001e8f2537e4f6cd104768a1214")}		
		
		fmt.Printf("Discovering services...\n")
		services, err := cln.DiscoverServices(uuidServiceFilter)
		if err != nil {
			fmt.Printf("can't discover services: %s \n", err)
			return
		}
		
		for _, s := range services {
			fmt.Printf("Discovering characteristics...\n")
			characteristics, err := cln.DiscoverCharacteristics(uuidCharacteristicsFilter, s)
			if err != nil {
				fmt.Printf("can't discover characteristics: %s \n", err)
				return
			}
			for _, c := range characteristics {
				fmt.Printf("      Characteristic: %s %s, Property: 0x%02X (%s), Handle(0x%02X), VHandle(0x%02X)\n",
				c.UUID, ble.Name(c.UUID), c.Property, propString(c.Property), c.Handle, c.ValueHandle)
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
		*/
