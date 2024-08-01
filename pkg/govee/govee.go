package govee

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/go-ble/ble"
)

// Data contains the data read from the sensor as well as a timestamp.
type Data struct {
        LocalName    string
        Address      ble.Addr
        RSSI         int
        LastUpdated  time.Time
        Temperature  float64
        Humidity     float64
        Battery      uint8
}

func ParseAdv(a ble.Advertisement) (*Data) {
    fmt.Printf("%s - ", a.LocalName())
    if a.Connectable() {
        fmt.Printf("[%s] C %3d:", a.Addr(), a.RSSI())
    } else {
        fmt.Printf("[%s] N %3d:", a.Addr(), a.RSSI())
    }
    comma := ""
    if len(a.LocalName()) > 0 {
        fmt.Printf(" Name: %s", a.LocalName())
        comma = ","
    }
    if len(a.Services()) > 0 {
        fmt.Printf("%s Svcs: %v", comma, a.Services())
        comma = ","
    }
    if len(a.ManufacturerData()) > 0 {
        fmt.Printf("%s MD: %X", comma, a.ManufacturerData())
    }
    fmt.Printf("\n")

    p := bytes.NewBuffer(a.ManufacturerData())

    var garbage int16
    if err := binary.Read(p, binary.BigEndian, &garbage); err != nil {
        fmt.Errorf("error reading data: %s", err)
        return nil
    }

    var t int32
    if err := binary.Read(p, binary.BigEndian, &t); err != nil {
        fmt.Errorf("error reading data: %s", err)
        return nil
    }

    var batt uint8
    if err := binary.Read(p, binary.BigEndian, &batt); err != nil {
        fmt.Errorf("error reading data: %s", err)
        return nil
    }

    neg := false
    if (0x800000 & t == 0x800000) {
        t = t & 0xFFFFF
        neg = true
    }

    Temperature := float64(math.Trunc(float64(t) / 1000)) / 10
    if (neg) {
        Temperature = -1 * Temperature
    }
    Humidity := math.Mod(float64(t),1000) / 10
    fmt.Printf("%s - Temp: %0.1f - Humidity: %0.1f%% - Batt: %d%%\n", time.Now().Format("2006-01-02 15:04:05"), Temperature, Humidity, batt)

        var data Data
        data.LocalName = a.LocalName()
        data.Address = a.Addr()
        data.RSSI = a.RSSI()
        data.LastUpdated = time.Now()
        data.Temperature = Temperature
        data.Humidity = Humidity
        data.Battery = batt

    return &data
}

