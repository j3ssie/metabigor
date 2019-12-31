package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// InitConfig Init the config
func InitConfig(options Options) {
	options.ConfigFile, _ = homedir.Expand(options.ConfigFile)
	RootFolder := filepath.Dir(options.ConfigFile)
	if !FolderExists(RootFolder) {
		InforF("Init new config at %v", RootFolder)
		os.MkdirAll(RootFolder, 0750)
	}

	// init config
	v := viper.New()
	v.AddConfigPath(RootFolder)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if !FileExists(options.ConfigFile) {
		InforF("Write new config to: %v", options.ConfigFile)
		v.SetDefault("Sessions", map[string]string{
			"fofa":    "xxx",
			"censys":  "xxx",
			"zoomeye": "xxx",
			"github":  "xxx",
		})
		v.SetDefault("Credentials", map[string]string{
			"fofa":    "username:password",
			"censys":  "username:password",
			"zoomeye": "username:password",
			"github":  "username:password",
		})
		v.WriteConfigAs(options.ConfigFile)
	} else {
		if options.Debug {
			InforF("Load config from: %v", options.ConfigFile)
		}
		b, _ := ioutil.ReadFile(options.ConfigFile)
		v.ReadConfig(bytes.NewBuffer(b))
	}
}

// GetCred get credentials
func GetCred(source string, options Options) string {
	options.ConfigFile, _ = homedir.Expand(options.ConfigFile)
	RootFolder := filepath.Dir(options.ConfigFile)
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(RootFolder)
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Failed to read the configuration file: %s\n", err)
		InitConfig(options)
		return ""
	}

	Creds := v.GetStringMapString("Credentials")
	if Creds == nil {
		return ""
	}
	for k, v := range Creds {
		if strings.ToLower(k) == strings.ToLower(source) {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// GetSess get credentials
func GetSess(source string, options Options) string {
	options.ConfigFile, _ = homedir.Expand(options.ConfigFile)
	RootFolder := filepath.Dir(options.ConfigFile)
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(RootFolder)
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Failed to read the configuration file: %s\n", err)
		InitConfig(options)
		return ""
	}
	Creds := v.GetStringMapString("Sessions")
	if Creds == nil {
		return ""
	}
	for k, v := range Creds {
		if strings.ToLower(k) == strings.ToLower(source) {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// SaveSess get credentials
func SaveSess(source string, sess string, options Options) string {
	options.ConfigFile, _ = homedir.Expand(options.ConfigFile)
	RootFolder := filepath.Dir(options.ConfigFile)
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(RootFolder)
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Failed to read the configuration file: %s\n", err)
		InitConfig(options)
		return ""
	}

	Sessions := v.GetStringMapString("Sessions")
	Sessions[source] = sess
	v.Set("Sessions", Sessions)
	v.WriteConfig()
	return sess
}
