package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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

//BleName get ble name from api
type BleName struct {
	Ble string `json:"ble"`
}

//WifiSettingStatus is the network health staus
type WifiSettingStatus struct {
	Success bool `json:"success"`
}

var (
	// serviceUUID = bluetooth.ServiceUUIDNordicUART
	// rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	// txUUID      = bluetooth.CharacteristicUUIDUARTTX
	serviceUUID, _ = bluetooth.ParseUUID("d6cb1959-8010-43bd-8ef7-48dbd249b984")
	rxUUID, _      = bluetooth.ParseUUID("493ebfb0-b690-4ae8-a77a-329619c6f613")
	txUUID, _      = bluetooth.ParseUUID("2d75504c-b822-44b3-bb81-65d7b6cbdae2")
	ipUUID, _      = bluetooth.ParseUUID("2d75504c-b822-44b3-bb81-65d7b6cbdae1")
	readUUID, _    = bluetooth.ParseUUID("2d75504c-b822-44b3-bb81-65d7b6cbdae3")
	settingUUID, _ = bluetooth.ParseUUID("2d75504c-b822-44b3-bb81-65d7b6cbdae4")
)

var localAPIHost = "http://127.0.0.1:3002/"

var getLocalIPAddressURL = localAPIHost + "getLocalIPAddress"
var internetHealthyCheckURL = localAPIHost + "internetHealthyCheck"
var ideviceIsInitializedURL = localAPIHost + "deviceIsInitialized"
var getBleServiceNameURL = localAPIHost + "getBleServiceName"
var setupNewWifiURL = localAPIHost + "setupNewWifi"

func main() {
	println("starting")
	adapter := bluetooth.DefaultAdapter
	must("enable BLE stack", adapter.Enable())
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "kimacloud-x11", // Nordic UART Service
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
	}))
	must("start adv", adv.Start())

	var rxChar bluetooth.Characteristic
	var txChar bluetooth.Characteristic
	var ipChar bluetooth.Characteristic
	var readChar bluetooth.Characteristic
	var settingChar bluetooth.Characteristic
	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &rxChar,
				UUID:   rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					txChar.Write(value)
				},
			},
			{
				Handle: &settingChar,
				UUID:   rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					txChar.Write(value)
				},
			},
			{
				Handle: &readChar,
				UUID:   readUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					// txChar.Write(value)

					ipaddresses, _ := getLocalIPAddresses()
					ipString, _ := json.Marshal(ipaddresses)
					log.Println(ipString)
					ipChar.Write(ipString)
				},
			},
			{
				Handle: &txChar,
				UUID:   txUUID,
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
	for {
		println("Go Bluetooth test /", address.MAC.String())
		time.Sleep(10 * time.Second)
	}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

func getLocalIPAddresses() (DeviceIPAddress, error) {
	log.Println("api to fetch the ip address")
	res, err := http.Get(getLocalIPAddressURL)
	if err != nil {
		log.Println(err)
		return DeviceIPAddress{}, err
	}

	defer res.Body.Close()
	rbody, _ := ioutil.ReadAll(res.Body)
	ipaddresses := DeviceIPAddress{}
	log.Println("--")
	log.Println(rbody)

	err = json.Unmarshal(rbody, &ipaddresses)
	if err != nil {
		log.Println(err)
		return DeviceIPAddress{}, err
	}
	log.Println(ipaddresses)
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
	log.Println(networkStatus)
	return networkStatus.Success, nil
}

func getBleServiceName() (string, error) {
	res, err := http.Get(getBleServiceNameURL)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer res.Body.Close()
	rbody, _ := ioutil.ReadAll(res.Body)
	bleName := BleName{}
	err = json.Unmarshal(rbody, &bleName)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println(bleName)
	return bleName.Ble, nil
}

func setupNewWifi(wifiConfig WIFIConfig) (bool, error) {
	postData, _ := json.Marshal(wifiConfig)

	request, _ := http.NewRequest("POST", setupNewWifiURL, bytes.NewBuffer(postData))
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
