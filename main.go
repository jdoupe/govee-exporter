package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/go-ble/ble"
        "github.com/go-ble/ble/examples/lib/dev"
        "github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/jdoupe/govee-exporter/internal/collector"
	"github.com/jdoupe/govee-exporter/internal/config"
	"github.com/jdoupe/govee-exporter/pkg/govee"
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

	mutex = &sync.Mutex{}
)

var	c = &collector.GoveeAllData{
		Data: make(map[string]*govee.Data),
		Log: log,
	}

func main() {
	config, err := config.Parse(log)
	if err != nil {
		log.Fatalf("Error in configuration: %s", err)
	}

	log.SetLevel(logrus.Level(config.LogLevel))
	log.Infof("Bluetooth Device: %s", config.Device)

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
	d, err := dev.NewDevice(config.Device)
        if err != nil {
                log.Fatalf("can't new device : %s", err)
        }
        ble.SetDefaultDevice(d)

        // Scan until interrupted by user.
	log.Info("Exporter is started.")
        chkErr(ble.Scan(ctx, true, advHandler, nil))

	wg.Wait()
	log.Info("Shutdown complete.")
}

func startSignalHandler(ctx context.Context, wg *sync.WaitGroup, cancel func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		log.Debug("Signal handler ready.")
		<-sigCh
		log.Debug("Got shutdown signal.")
		signal.Reset()
		cancel()
	}()
}

func advHandler(a ble.Advertisement) {
    log.Tracef("[%s] %3d: Name: %s, Svcs: %v, MD: %X", a.Addr(), a.RSSI(), a.LocalName(), a.Services(), a.ManufacturerData())
    if strings.HasPrefix(a.LocalName(),"GV") {
		mutex.Lock()
        c.Data[a.LocalName()] = govee.ParseAdv(a,log)
		mutex.Unlock()
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
