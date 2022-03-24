package alice

import "fmt"

type Config struct {
	UseTrackers          bool
	UseDHT               bool
	ShowDownloadProgress bool
}

var DefaultConfig = Config{
	UseTrackers:          true,
	UseDHT:               true,
	ShowDownloadProgress: true,
}

func NewConfig(config Config) error {
	if !config.UseTrackers && !config.UseDHT {
		err := fmt.Errorf("enable tracker or dht peer discovery")
		return err
	}
	DefaultConfig = config
	return nil
}
