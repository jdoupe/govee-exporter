package govee

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/go-ble/ble"
        "github.com/sirupsen/logrus"
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

func ParseAdv(a ble.Advertisement, log *logrus.Logger) (*Data) {
    log.Debugf("[%s] %3d: Name: %s, Svcs: %v, MD: %X", a.Addr(), a.RSSI(), a.LocalName(), a.Services(), a.ManufacturerData())

    p := bytes.NewBuffer(a.ManufacturerData())

    var garbage int16
    if err := binary.Read(p, binary.BigEndian, &garbage); err != nil {
        log.Errorf("error reading data: %s", err)
        return nil
    }

    var t int32
    if err := binary.Read(p, binary.BigEndian, &t); err != nil {
        log.Errorf("error reading data: %s", err)
        return nil
    }

    var batt uint8
    if err := binary.Read(p, binary.BigEndian, &batt); err != nil {
        log.Errorf("error reading data: %s", err)
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
    log.Debugf("%s - Temp: %0.1f - Humidity: %0.1f%% - Batt: %d%%\n", time.Now().Format("2006-01-02 15:04:05"), Temperature, Humidity, batt)

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

