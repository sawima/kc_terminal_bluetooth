package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"tinygo.org/x/bluetooth"
)

//DeviceIPAddress model
type DeviceIPAddress struct {
	Eth0 IPModel `json:"eth0"`
	Wifi IPModel `json:"wifi"`
}

//IPModel for ip address
type IPModel struct {
	IP   string `json:"ip"`
	Mac  string `json:"mac"`
	Name string `json:"name"`
}

//WIFIConfig data send to config device wifi
type WIFIConfig struct {
	Ssid     string `json:"ssid"`
	Password string `json:"password"`
}

//NetworkStatus is the network health staus
type NetworkStatus struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

//WifiSettingStatus is the network health staus
type WifiSettingStatus struct {
	Success bool `json:"success"`
}

var (
	serviceUUID, _       = bluetooth.ParseUUID("d6cb1959-8010-43bd-8ef7-48dbd249b984")
	refreshUUID, _       = bluetooth.ParseUUID("c537baa5-6201-4275-ab14-da353bde3dc3")
	statusUUID, _        = bluetooth.ParseUUID("f9e9e098-77d4-4db3-a08f-8321c493431b")
	ipUUID, _            = bluetooth.ParseUUID("2d75504c-b822-44b3-bb81-65d7b6cbdae1")
	settingUUID, _       = bluetooth.ParseUUID("493ebfb0-b690-4ae8-a77a-329619c6f613")
	resetTerminalUUID, _ = bluetooth.ParseUUID("2d75504c-b822-44b3-bb81-65d7b6cbdae3")
)

var localAPIHost = "http://127.0.0.1:3002/"
var getLocalIPAddressURL = localAPIHost + "getLocalIPAddress"
var internetHealthyCheckURL = localAPIHost + "internetHealthyCheck"
var setupNewWifiURL = localAPIHost + "setupNewWifi"
var factoryResetURL = localAPIHost + "factoryResetForBle"

var deviceBleFile = "/application/signage-device-application/db/device.txt"

func main() {
	file, err := ioutil.ReadFile(deviceBleFile)
	if err != nil {
		log.Println(err)
	}
	bleName := string(file)
	adapter := bluetooth.DefaultAdapter
	must("enable BLE stack", adapter.Enable())
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    bleName, // kimacloud sevice
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
	}))
	must("start adv", adv.Start())
	var refreshChar bluetooth.Characteristic
	var statusChar bluetooth.Characteristic
	var ipChar bluetooth.Characteristic
	var resetTerminalChar bluetooth.Characteristic
	var settingChar bluetooth.Characteristic
	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &refreshChar,
				UUID:   refreshUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					ipaddresses, _ := getLocalIPAddresses()
					ipString, _ := json.Marshal(ipaddresses)
					ipChar.Write(ipString)
					netState, _ := internetHealthyCheck()
					if netState {
						statusChar.Write([]byte("online"))
					} else {
						statusChar.Write([]byte("offline"))
					}

				},
			},
			{
				Handle: &settingChar,
				UUID:   settingUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					setupNewWifi(value)
					ipaddresses, _ := getLocalIPAddresses()
					ipString, _ := json.Marshal(ipaddresses)
					log.Println(ipString)
					netState, _ := internetHealthyCheck()
					if netState {
						statusChar.Write([]byte("online"))
					} else {
						statusChar.Write([]byte("offline"))
					}

				},
			},
			{
				Handle: &resetTerminalChar,
				UUID:   resetTerminalUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					log.Println("reset terminal")
					resetTerminal(value)
				},
			},
			{
				Handle: &statusChar,
				UUID:   statusUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
			{
				Handle: &ipChar,
				UUID:   ipUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	}))
	println("advertising...")
	ipaddresses, _ := getLocalIPAddresses()
	ipString, _ := json.Marshal(ipaddresses)
	log.Println(ipString)
	ipChar.Write(ipString)
	address, _ := adapter.Address()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		println("Kimacloud Bluetooth Service /", address.MAC.String())
		time.Sleep(1 * time.Second)
	}()
	<-c
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

func getLocalIPAddresses() (DeviceIPAddress, error) {
	res, err := http.Get(getLocalIPAddressURL)
	if err != nil {
		log.Println(err)
		return DeviceIPAddress{}, err
	}

	defer res.Body.Close()
	rbody, _ := ioutil.ReadAll(res.Body)
	ipaddresses := DeviceIPAddress{}

	err = json.Unmarshal(rbody, &ipaddresses)
	if err != nil {
		log.Println(err)
		return DeviceIPAddress{}, err
	}
	return ipaddresses, nil
}

func internetHealthyCheck() (bool, error) {
	res, err := http.Get(internetHealthyCheckURL)
	if err != nil {
		log.Println(err)
		return false, err
	}
	defer res.Body.Close()
	rbody, _ := ioutil.ReadAll(res.Body)
	networkStatus := NetworkStatus{}
	err = json.Unmarshal(rbody, &networkStatus)
	if err != nil {
		log.Println(err)
		return false, err
	}
	return networkStatus.Success, nil
}

func setupNewWifi(wifiConfig []byte) (bool, error) {
	request, _ := http.NewRequest("POST", setupNewWifiURL, bytes.NewBuffer(wifiConfig))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	res, err := client.Do(request)

	if err != nil {
		log.Println(err)
		return false, err
	}
	defer res.Body.Close()
	rbody, _ := ioutil.ReadAll(res.Body)
	wifiSettingStatus := WifiSettingStatus{}
	err = json.Unmarshal(rbody, &wifiSettingStatus)
	if err != nil {
		log.Println(err)
		return false, err
	}
	log.Println(wifiSettingStatus)
	return wifiSettingStatus.Success, nil
}

func resetTerminal(resetVersion []byte) (bool, error) {
	request, _ := http.NewRequest("POST", factoryResetURL, bytes.NewBuffer(resetVersion))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	res, err := client.Do(request)

	if err != nil {
		log.Println(err)
		return false, err
	}
	defer res.Body.Close()
	return true, nil
}
