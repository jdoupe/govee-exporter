package collector

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/jdoupe/govee-exporter/internal/config"
	"github.com/jdoupe/govee-exporter/pkg/govee"
)

const (
	// MetricPrefix contains the prefix used by all metrics emitted from this collector.
	MetricPrefix = "govee_"
)

var (
	varLabelNames = []string{
		"macaddress",
		"name",
	}

	upDesc = prometheus.NewDesc(
		MetricPrefix+"up",
		"Shows if data could be successfully retrieved by the collector.",
		varLabelNames, nil)
	updatedTimestampDesc = prometheus.NewDesc(
		MetricPrefix+"updated_timestamp",
		"Contains the timestamp when the last communication with the Bluetooth device happened.",
		varLabelNames, nil)
	batteryDesc = prometheus.NewDesc(
		MetricPrefix+"battery_percent",
		"Battery level in percent.",
		varLabelNames, nil)
	temperatureDesc = prometheus.NewDesc(
		MetricPrefix+"temperature_celsius",
		"Ambient temperature in celsius.",
		varLabelNames, nil)
	humidityDesc = prometheus.NewDesc(
		MetricPrefix+"humidity_percent",
		"Ambient humidity in percent.",
		varLabelNames, nil)
)

// Govee implements a Prometheus collector that emits metrics of a Govee thermometer/hygrometer.
type Govee struct {
	Log           logrus.FieldLogger
	Source        func(macAddress string) (govee.Data, error)
	Sensors       []config.Sensor
	StaleDuration time.Duration
}

// Describe implements prometheus.Collector
func (c *Govee) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- updatedTimestampDesc
	ch <- batteryDesc
	ch <- temperatureDesc
	ch <- humidityDesc
}

// Collect implements prometheus.Collector
func (c *Govee) Collect(ch chan<- prometheus.Metric) {
	for _, s := range c.Sensors {
		c.collectSensor(ch, s)
	}
}

func (c *Govee) collectSensor(ch chan<- prometheus.Metric, s config.Sensor) {
	labels := []string{
		s.MacAddress,
		s.Name,
	}

	data, err := c.Source(s.MacAddress)
	if err != nil {
		c.Log.Errorf("Error getting data for %q: %s", s, err)
		c.sendMetric(ch, upDesc, 0, labels)

		return
	}
	c.sendMetric(ch, upDesc, 1, labels)
	c.sendMetric(ch, updatedTimestampDesc, float64(data.Time.Unix()), labels)

	age := time.Since(data.Time)
	if age >= c.StaleDuration {
		c.Log.Debugf("Data for %q is stale: %s > %s", s, age, c.StaleDuration)
		return
	}

	c.collectData(ch, data, labels)
}

func (c *Govee) collectData(ch chan<- prometheus.Metric, data govee.Data, labels []string) {
	for _, metric := range []struct {
		Desc  *prometheus.Desc
		Value float64
	}{
		{
			Desc:  batteryDesc,
			Value: float64(data.Sensors.Battery),
		},
		{
			Desc:  temperatureDesc,
			Value: float64(data.Sensors.Temperature),
		},
		{
			Desc:  humidityDesc,
			Value: float64(data.Sensors.Humidity),
		},
	} {
		c.sendMetric(ch, metric.Desc, metric.Value, labels)
	}
}

func (c *Govee) sendMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels []string) {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, labels...)
	if err != nil {
		c.Log.Errorf("can not create metric %q: %s", desc, err)
		return
	}

	ch <- m
}
