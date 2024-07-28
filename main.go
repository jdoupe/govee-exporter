package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-ble/ble"
        "github.com/go-ble/ble/examples/lib/dev"
        "github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/jdoupe/govee-exporter/internal/collector"
	"github.com/jdoupe/govee-exporter/internal/config"
//	"github.com/jdoupe/govee-exporter/internal/updater"
)

var (
	log = &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableTimestamp: true,
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}

	version = "dev"
	commit  = "none"
	date    = "unknown"
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

var allData = make(map[string]*Data)

func main() {
	config, err := config.Parse(log)
	if err != nil {
		log.Fatalf("Error in configuration: %s", err)
	}

	log.SetLevel(logrus.Level(config.LogLevel))
	log.Infof("Bluetooth Device: %s", config.Device)

	/*
	provider, err := updater.New(log, config.Device, config.RefreshTimeout, config.Retry)
	if err != nil {
		log.Fatalf("Error creating device: %s", err)
	}

	for _, s := range config.Sensors {
		log.Infof("Sensor: %s", s)
		provider.AddSensor(s)
	}
	*/

	c := &collector.Govee{
		Log:           log,
//		Source:        provider.GetData,
		Sensors:       config.Sensors,
		StaleDuration: config.StaleDuration,
	}
	if err := prometheus.Register(c); err != nil {
		log.Fatalf("Failed to register collector: %s", err)
	}

	versionMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: collector.MetricPrefix + "build_info",
		Help: "Contains build information as labels. Value set to 1.",
		ConstLabels: prometheus.Labels{
			"version": version,
			"commit":  commit,
			"date":    date,
		},
	})
	versionMetric.Set(1)
	prometheus.MustRegister(versionMetric)

	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))

	go func() {
		log.Infof("Listen on %s...", config.ListenAddr)
		log.Fatal(http.ListenAndServe(config.ListenAddr, nil))
	}()

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	startSignalHandler(ctx, wg, cancel)
	/*
	startScheduleLoop(ctx, wg, config, provider)
	provider.Start(ctx, wg)
	*/
	d, err := dev.NewDevice(config.Device)
        if err != nil {
                log.Fatalf("can't new device : %s", err)
        }
        ble.SetDefaultDevice(d)

        // Scan until interrupted by user.
        //fmt.Printf("Scanning for %s...\n", *du)
        //ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *du))
	log.Info("Exporter is started.")
        chkErr(ble.Scan(ctx, true, advHandler, nil))

	wg.Wait()
	log.Info("Shutdown complete.")
}

func startSignalHandler(ctx context.Context, wg *sync.WaitGroup, cancel func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		log.Debug("Signal handler ready.")
		<-sigCh
		log.Debug("Got shutdown signal.")
		signal.Reset()
		cancel()
	}()
}

/*
func startScheduleLoop(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, provider *updater.Updater) {
	wg.Add(1)

	refresher := time.NewTicker(cfg.RefreshDuration)
	provider.UpdateAll(time.Now())

	go func() {
		defer wg.Done()

		log.Debug("Schedule loop ready.")
		for {
			select {
			case <-ctx.Done():
				log.Debug("Shutting down refresh loop")
				return
			case now := <-refresher.C:
				log.Debugf("Updating all at %s", now)
				provider.UpdateAll(now)
			}
		}
	}()
}
*/

func advHandler(a ble.Advertisement) {
    if strings.HasPrefix(a.LocalName(),"GV") {
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
                return
        }

        var t int32
        if err := binary.Read(p, binary.BigEndian, &t); err != nil {
                fmt.Errorf("error reading data: %s", err)
                return
        }

        var batt uint8
        if err := binary.Read(p, binary.BigEndian, &batt); err != nil {
                fmt.Errorf("error reading data: %s", err)
                return
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

	if allData[a.LocalName()] == nil {
		var data Data
		data.LocalName = a.LocalName()
		data.Address = a.Addr()
		data.RSSI = a.RSSI()
		data.LastUpdated = time.Now()
		data.Temperature = Temperature
		data.Humidity = Humidity
		data.Battery = batt
		allData[a.LocalName()] = &data
	} else {
		allData[a.LocalName()].RSSI = a.RSSI()
		allData[a.LocalName()].RSSI = a.RSSI()
		allData[a.LocalName()].LastUpdated = time.Now()
		allData[a.LocalName()].Temperature = Temperature
		allData[a.LocalName()].Humidity = Humidity
		allData[a.LocalName()].Battery = batt
	}
	fmt.Print(allData)
    }
}

func chkErr(err error) {
        switch errors.Cause(err) {
        case nil:
        case context.DeadlineExceeded:
                fmt.Printf("done\n")
        case context.Canceled:
                fmt.Printf("canceled\n")
        default:
                log.Fatalf(err.Error())
        }
}
