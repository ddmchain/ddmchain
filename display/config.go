
package dashboard

import "time"

var DefaultConfig = Config{
	Host:    "localhost",
	Port:    8080,
	Refresh: 5 * time.Second,
}

type Config struct {

	Host string `toml:",omitempty"`

	Port int `toml:",omitempty"`

	Refresh time.Duration `toml:",omitempty"`

	Assets string `toml:",omitempty"`
}
