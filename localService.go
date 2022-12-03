package main

// import (
// 	"encoding/json"
// 	"io/ioutil"
// 	"log"
// 	"net/http"
// )

// var localAPIHost = "http://127.0.0.1:3002/"

// var getLocalIPAddressURL = localAPIHost + "getLocalIPAddress"
// var internetHealthyCheckURL = localAPIHost + "internetHealthyCheck"
// var ideviceIsInitializedURL = localAPIHost + "deviceIsInitialized"
// var getBleServiceNameURL = localAPIHost + "getBleServiceName"
// var setupNewWifiURL = localAPIHost + "setupNewWifi"

// //GetLocalIPAddresses to fetch ip address from local api
// func getLocalIPAddresses() (DeviceIPAddress, error) {
// 	res, err := http.Get(getLocalIPAddressURL)
// 	if err != nil {
// 		log.Println(err)
// 		return DeviceIPAddress{}, err
// 	}

// 	defer res.Body.Close()
// 	rbody, _ := ioutil.ReadAll(res.Body)
// 	ipaddresses := DeviceIPAddress{}
// 	err = json.Unmarshal(rbody, &ipaddresses)
// 	if err != nil {
// 		log.Println(err)
// 		return DeviceIPAddress{}, err
// 	}
// 	return ipaddresses, nil
// }
