package config

import (
	"github.com/spf13/viper"
	"os"
	"strings"
)

type GossipConfig struct {
	CacheSize        int      `mapstructure:"cache_size"`
	Degree           int      `mapstructure:"degree"`
	MinConnections   int      `mapstructure:"minConnections"`
	MaxConnections   int      `mapstructure:"maxConnections"`
	Bootstrapper     string   `mapstructure:"bootstrapper"`
	P2PAddress       string   `mapstructure:"p2pAddress"`
	APIAddress       string   `mapstructure:"apiAddress"`
	KnownPeers       []string `mapstructure:"knownPeers"`
	MaintainInterval int      `mapstructure:"maintainInterval"`
}

type LogConfig struct {
	Level   string `mapstructure:"level"`
	LogFile string `mapstructure:"logFile"`
}

// LoadConfig load config via viper from file from the path
// Input config path will indicate the path of the config file
func LoadConfig(configPath string) (*GossipConfig, *LogConfig, error) {
	//check whether the file exits
	var P2PConfig *GossipConfig
	var LoggerConfig *LogConfig

	fileExists := fileExists(configPath)
	if !fileExists {
		return nil, nil, os.ErrNotExist
	}
	// open the file with configPath
	viper.SetConfigFile(configPath)
	// read the file
	err := viper.ReadInConfig()
	if err != nil {
		return nil, nil, err
	}
	P2PConfig = &GossipConfig{}
	LoggerConfig = &LogConfig{}
	//unmarshal the config file to the struct
	err = viper.UnmarshalKey("gossip", P2PConfig)
	if err != nil {
		return nil, nil, err
	}

	//clear all " " at beginning and end of the string
	P2PConfig.Bootstrapper = strings.TrimSpace(P2PConfig.Bootstrapper)
	P2PConfig.P2PAddress = strings.TrimSpace(P2PConfig.P2PAddress)
	P2PConfig.APIAddress = strings.TrimSpace(P2PConfig.APIAddress)
	//check whether the knownPeers is empty
	for i := 0; i < len(P2PConfig.KnownPeers); i++ {
		P2PConfig.KnownPeers[i] = strings.TrimSpace(P2PConfig.KnownPeers[i])
	}
	//check some config parameter
	if P2PConfig.CacheSize <= P2PConfig.Degree {
		return nil, nil, os.ErrInvalid
	} else if P2PConfig.MinConnections > P2PConfig.MaxConnections {
		return nil, nil, os.ErrInvalid
	} else if P2PConfig.Degree < 1 {
		return nil, nil, os.ErrInvalid
	} else if P2PConfig.Bootstrapper == "" { //the bootstrapper is the first peer we connect to
		return nil, nil, os.ErrInvalid
	} else if P2PConfig.P2PAddress == "" { //the p2p address is the address we listen to
		return nil, nil, os.ErrInvalid
	} else if P2PConfig.APIAddress == "" { //the api address is the address we listen to
		return nil, nil, os.ErrInvalid
	}

	err = viper.UnmarshalKey("log", LoggerConfig)
	if err != nil {
		return nil, nil, err
	}
	return P2PConfig, LoggerConfig, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
