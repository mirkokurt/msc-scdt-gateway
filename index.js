var noble = require('noble');
var bleno = require('bleno');


noble.on('stateChange', function(state) {
  if (state === 'poweredOn') {
    noble.startScanning([], true);
  } else {
    noble.stopScanning();
  }
});

bleno.on('stateChange', function(state) {
  if (state === 'poweredOn') {
    bleno.startAdvertising('Contact Gateway',['ff00']);
  } 
});

noble.on('discover', function(peripheral) {

if (peripheral.advertisement.localName === "SyncCont") {
  console.log('\thello my local name is:');
  console.log('\t\t' + peripheral.advertisement.localName);

  if (peripheral.advertisement.manufacturerData) {
    console.log('\there is my manufacturer data:');
    console.log('\t\t' + JSON.stringify(peripheral.advertisement.manufacturerData.toString('hex')));
  }

  console.log();

}

});