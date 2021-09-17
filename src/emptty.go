package src

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const version = "0.5.0"

var buildVersion string

// Main handles the functionality of whole application.
func Main() {
	//Server()
	if contains(os.Args, "-h") || contains(os.Args, "--help") {
		printHelp()
		os.Exit(0)
	}
	if contains(os.Args, "-v") || contains(os.Args, "--version") {
		fmt.Printf("emptty %s\nhttps://github.com/tvrzna/emptty\n\nReleased under the MIT License.\n", getVersion())
		os.Exit(0)
	}

	conf := newDefaultConfig()
	err := loadConfig(conf, pathConfigFile)
	if err != nil {
		os.Exit(0)
	}

	err = loadConfigDir(conf, pathConfigDir)
	if err != nil {
		os.Exit(0)
	}

	for i, arg := range os.Args {
		switch arg {
		case "-t", "--tty":
			if len(os.Args) > i+1 {
				tty := parseTTY(os.Args[i+1], "0")
				if tty > 0 {
					conf.minTty = tty
				}
			}
		case "-d", "--daemon":
			conf.daemonMode = true
		}
	}

	//var fTTY *os.File
	//if conf.daemonMode {
	//	fTTY = startDaemon(conf)
	//}

	initLogger(conf)
	log.Println("emptty start")
	printMotd(conf)
	log.Println("motd end")
	login(conf)

	//if conf.daemonMode {
	//	stopDaemon(conf, fTTY)
	//}
}

// child-session process
// emptty --> child-session --> greeter/session entry
func Child() error {
	return nil
}

// Prints help
func printHelp() {
	fmt.Println("Usage: emptty [options]")
	fmt.Println("Options:")
	fmt.Printf("  -h, --help\t\tprint this help\n")
	fmt.Printf("  -v, --version\t\tprint version\n")
	fmt.Printf("  -d, --daemon\t\tstart in daemon mode\n")
	fmt.Printf("  -t, --tty NUMBER\toverrides configured TTY number\n")
}

// Gets current version
func getVersion() string {
	tags := strings.Builder{}
	for _, tag := range []string{tagPam, tagUtmp, tagXlib} {
		if tags.Len() > 0 {
			tags.WriteString(", ")
		}
		tags.WriteString(tag)
	}
	if buildVersion != "" {
		if tags.Len() == 0 {
			return buildVersion[1:]
		}
		return buildVersion[1:] + " (" + tags.String() + ")"
	}
	return version
}
