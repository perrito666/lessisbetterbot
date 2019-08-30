package main

import (
	"fmt"
	"log"
	"os"

	"github.com/juju/gnuflag"
	"github.com/pkg/errors"
)

var (
	ErrNoConfigPath = errors.New("config path not passed")
	ErrNoConfig     = errors.New("config file does not exist")
	ErrConfigExists = errors.New("config file exists")
)

func flags() (c *Config, err error) {
	gnuflag.CommandLine.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [flags] /path/to/config.ini:\n", os.Args[0])
		gnuflag.PrintDefaults()
	}
	createConfig := gnuflag.CommandLine.Bool("createconfig", false, "create the initial config on the config file, will only work if the file does not exist")
	netName := gnuflag.CommandLine.String("network", "freenode", "the name of the url to be connected to (must match config file section)")
	defer func() {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Found Errors: %v\n\n", err)
			gnuflag.CommandLine.Usage()
		}
	}()
	err = gnuflag.CommandLine.Parse(true, os.Args[1:])
	if err != nil {
		return nil, err
	}
	args := gnuflag.CommandLine.Args()
	if len(args) != 1 {
		return nil, ErrNoConfigPath
	}

	configPath := args[0]
	if *createConfig {
		if _, err := os.Stat(configPath); err == nil {
			return nil, ErrConfigExists
		}
		f, err := os.OpenFile(configPath, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			return nil, errors.Wrap(err, "creating config file")
		}
		defer f.Close()
		err = writeConfig(f, nil)
		if err != nil {
			return nil, errors.Wrap(err, "creating initial config")
		}
		return nil, nil
	} else {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, ErrNoConfig
		}
	}
	c, err = LoadConfig(configPath, *netName)
	if err != nil {
		return nil, errors.Wrapf(err, "loading config from %q for network %q", configPath, *netName)
	}
	return c, nil
}

func main() {
	cfg, err := flags()
	if err != nil {
		// flags will print its own stuff
		os.Exit(1)
	}
	if cfg == nil {
		return
	}

	// Logging initializing, no options, just vanilla
	logger := log.New(os.Stdout, "lessisbetterbot: ", log.Ldate|log.Ltime|log.Lshortfile)

	libb := bot{
		logger: logger,
		cfg:    cfg,
	}

	logger.Println("going live")
	logger.Println(cfg.NickName)
	logger.Println(cfg.Channels)

	fmt.Printf("%v\n", libb.live())

}
