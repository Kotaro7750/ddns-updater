package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/rdegges/go-ipify"
	"github.com/robfig/cron/v3"
)

type MyGlobalIPAddress struct {
	IPv4 net.IP
	IPv6 net.IP
}

type Config struct {
	CronExpression string `env:"CRON_EXPRESSION" envDefault:"*/5 * * * *"`
	ZoneName       string `env:"ZONE_NAME,required"`
	RecordName     string `env:"RECORD_NAME,required"`
	EnableIPv4     bool   `env:"ENABLE_IPV4" envDefault:"true"`
	EnableIPv6     bool   `env:"ENABLE_IPV6" envDefault:"true"`
}

var lastCheckedGlobalIPAddress = MyGlobalIPAddress{}
var recordUpdater = RecordUpdater{}
var config = Config{}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := env.Parse(&config); err != nil {
		slog.Error("Failed to parse environment variables", "error", err)
		return
	}

	slog.Info("Starting DDNS Updater", "zone", config.ZoneName, "record", config.RecordName, "cron", config.CronExpression, "ipv4_enabled", config.EnableIPv4, "ipv6_enabled", config.EnableIPv6)

	recordUpdater = RecordUpdater{
		ZoneName:   config.ZoneName,
		RecordName: config.RecordName,
	}

	PeriodicUpdateTask()

	c := cron.New()
	_, err := c.AddFunc(config.CronExpression, PeriodicUpdateTask)
	if err != nil {
		slog.Error("Failed to add cron job", "error", err)
		return
	}

	c.Start()
	slog.Info("Cron scheduler started")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	slog.Info("Received shutdown signal, stopping cron scheduler")

	c.Stop()
	slog.Info("DDNS Updater stopped")
}

func PeriodicUpdateTask() {
	slog.Info("Periodic update task triggered")

	addr, err := getGlobalIPAddress()
	if err != nil {
		slog.Error("Failed to get global IP address", "error", err)
		return
	}

	if config.EnableIPv4 {
		if !lastCheckedGlobalIPAddress.IPv4.Equal(addr.IPv4) {
			slog.Info("IPv4 address has changed", "old", lastCheckedGlobalIPAddress.IPv4.String(), "new", addr.IPv4.String())

			if err := recordUpdater.updateDNSRecord(addr.IPv4); err != nil {
				slog.Error("Failed to update DNS record", "error", err)
				return
			}

			lastCheckedGlobalIPAddress.IPv4 = addr.IPv4
		} else {
			slog.Info("IPv4 address has not changed. skipping update")
		}
	} else {
		slog.Info("IPv4 update is disabled. skipping update")
	}

	if config.EnableIPv6 {
		if !lastCheckedGlobalIPAddress.IPv6.Equal(addr.IPv6) {
			slog.Info("IPv6 address has changed", "old", lastCheckedGlobalIPAddress.IPv6.String(), "new", addr.IPv6.String())

			if err := recordUpdater.updateDNSRecord(addr.IPv6); err != nil {
				slog.Error("Failed to update DNS record", "error", err)
				return
			}

			lastCheckedGlobalIPAddress.IPv6 = addr.IPv6
		} else {
			slog.Info("IPv6 address has not changed. skipping update")
		}
	} else {
		slog.Info("IPv6 update is disabled. skipping update")
	}

	slog.Info("Periodic update task finished")
}

func getGlobalIPAddress() (MyGlobalIPAddress, error) {
	slog.Info("Getting global IP address")

	ipify.API_URI = "https://api.ipify.org"
	ipv4Str, err := ipify.GetIp()
	if err != nil {
		return MyGlobalIPAddress{}, err
	}

	ipv4 := net.ParseIP(ipv4Str)
	if ipv4 == nil {
		return MyGlobalIPAddress{}, fmt.Errorf("Cannot parse IPv4 address %s", ipv4Str)
	}

	ipify.API_URI = "https://api6.ipify.org"
	ipv6Str, err := ipify.GetIp()
	if err != nil {
		return MyGlobalIPAddress{}, err
	}

	ipv6 := net.ParseIP(ipv6Str)
	if ipv6 == nil {
		return MyGlobalIPAddress{}, fmt.Errorf("Cannot parse IPv6 address %s", ipv6Str)
	}

	slog.Info("Got global IP address", "ipv4", ipv4.String(), "ipv6", ipv6.String())

	return MyGlobalIPAddress{
		IPv4: ipv4,
		IPv6: ipv6,
	}, nil
}
