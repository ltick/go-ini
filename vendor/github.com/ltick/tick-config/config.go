package config

import (
	"github.com/ltick/tick-config/ini"
	libConfig "github.com/go-ozzo/ozzo-config"
    "encoding/json"
    "gopkg.in/ini.v1"
)

type Config struct{
    *libConfig.Config
}

// New creates a new Config object.
func New() *Config {
    return &Config{
        libConfig.New(),
    }
}

func init() {
	libConfig.UnmarshalFuncMap[".ini"] = ini.Unmarshal
}
