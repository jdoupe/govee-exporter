// Package govee provides a function to read data from govee sensors using Bluetooth LE.
package govee

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/go-ble/ble"
	"github.com/sirupsen/logrus"
)

var (
	sensorCharacteristic = &ble.Characteristic{
		ValueHandle: 0xec,
	}
)

// Data contains the data read from the sensor as well as a timestamp.
type Data struct {
	Time     time.Time
	Sensors  Sensors
}

// Sensors contains the sensor data.
type Sensors struct {
	Temperature  float64
	Humidity     float64
	Battery      uint16
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *Sensors) UnmarshalBinary(data []byte) error {
	//if len(data) != 16 {
	//	return fmt.Errorf("invalid data length: %d != 10", len(data))
	//}

	p := bytes.NewBuffer(data)
	var t int32

	if err := binary.Read(p, binary.BigEndian, &t); err != nil {
		return fmt.Errorf("error reading data: %s", err)
	}

	s.Temperature = float64(math.Trunc(float64(t) / 1000)) / 10
	s.Humidity = math.Mod(float64(t),1000) / 10
	return nil
}

// ReadData uses a Bluetooth LE device to read data from the sensor identified using the MAC address.
func ReadData(ctx context.Context, log logrus.FieldLogger, device ble.Device, macAddress string) (Data, error) {
	addr := ble.NewAddr(macAddress)
	log.Debugf("Connecting to %q...", macAddress)
	c, err := device.Dial(ctx, addr)
	if err != nil {
		return Data{}, fmt.Errorf("error dialing: %s", err)
	}
	
	sensorsRaw, err := c.ReadCharacteristic(sensorCharacteristic)
	if err != nil {
		return Data{}, fmt.Errorf("error reading sensor data: %s", err)
	}
	log.Debugf("sensorsRaw of %q: %#v", macAddress, sensorsRaw)

	var sensors Sensors
	if err := sensors.UnmarshalBinary(sensorsRaw); err != nil {
		return Data{}, fmt.Errorf("error parsing sensor data: %s", err)
	}
	log.Debugf("Sensors of %q: %#v", macAddress, sensors)

	return Data{
		Time:     time.Now(),
		Sensors:  sensors,
	}, nil
}
