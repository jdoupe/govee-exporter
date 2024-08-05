# govee-exporter

A [prometheus](https://prometheus.io) exporter which can read data from Govee Hygrometer devices using Bluetooth.

Currently only support model H5075.

## Installation

This assumes you already have a bluetooth adapter in your system, along with appropriate drivers loaded. This is included in many distributions. See https://www.bluez.org/.

First clone the repository, then run the following command to get a binary for your current operating system / architecture. This assumes a working Go installation with modules support (Go >= 1.12.0):

```bash
git clone https://github.com/jdoupe/govee-exporter.git
cd govee-exporter
go build .
```

## Usage

```plain
$ govee-exporter -h
Usage of ./govee-exporter:
  -i, --adapter string            Bluetooth device to use for communication. (default "hci0")
  -a, --addr string               Address to listen on for connections. (default ":9294")
      --log-level string          Log level. (default "info")
```

After starting the server will offer the metrics on the `/metrics` endpoint, which can be used as a target for prometheus.

