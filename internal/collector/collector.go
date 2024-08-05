package collector

import (

	"github.com/prometheus/client_golang/prometheus"
	"github.com/jdoupe/govee-exporter/pkg/govee"
        "github.com/sirupsen/logrus"
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

type GoveeAllData struct {
    Data map[string]*govee.Data
    Log *logrus.Logger
}

// Describe implements prometheus.Collector
func (c *GoveeAllData) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- updatedTimestampDesc
	ch <- batteryDesc
	ch <- temperatureDesc
	ch <- humidityDesc
}

// Collect implements prometheus.Collector
func (c *GoveeAllData) Collect(ch chan<- prometheus.Metric) {
  c.Log.Debugf("Sending data...\n")
  for _, d := range c.Data {
      labels := []string{
        d.Address.String(),
        d.LocalName,
      }
      c.sendMetric(ch, upDesc, 1, labels)
      c.sendMetric(ch, updatedTimestampDesc, float64(d.LastUpdated.Unix()), labels)

        for _, metric := range []struct {
                Desc  *prometheus.Desc
                Value float64
        }{
                {
                        Desc:  batteryDesc,
                        Value: float64(d.Battery),
                },
                {
                        Desc:  temperatureDesc,
                        Value: float64(d.Temperature),
                },
                {
                        Desc:  humidityDesc,
                        Value: float64(d.Humidity),
                },
        } {
                c.sendMetric(ch, metric.Desc, metric.Value, labels)
        }

  }

}

func (c *GoveeAllData) sendMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels []string) {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, labels...)
	if err != nil {
		c.Log.Errorf("can not create metric %q: %s", desc, err)
		return
	}

	ch <- m
}
