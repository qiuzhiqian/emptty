package src

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	ini "gopkg.in/ini.v1"
)

const (
	confMinTTY             = "MIN_TTY"
	confSwitchTTY          = "SWITCH_TTY"
	confPrintIssue         = "PRINT_ISSUE"
	confDefaultUser        = "DEFAULT_USER"
	confAutologin          = "AUTOLOGIN"
	confAutologinSession   = "AUTOLOGIN_SESSION"
	confLang               = "LANG"
	confDbusLaunch         = "DBUS_LAUNCH"
	confXinitrcLaunch      = "XINITRC_LAUNCH"
	confVerticalSelection  = "VERTICAL_SELECTION"
	confLogging            = "LOGGING"
	confXorgArgs           = "XORG_ARGS"
	confLoggingFile        = "LOGGING_FILE"
	confDynamicMotd        = "DYNAMIC_MOTD"
	confFgColor            = "FG_COLOR"
	confBgColor            = "BG_COLOR"
	confDisplayStartScript = "DISPLAY_START_SCRIPT"
	confDisplayStopScript  = "DISPLAY_STOP_SCRIPT"

	pathConfigFile = "/etc/emptty.conf"
	pathConfigDir  = "/etc/emptty.d"

	constLogDefault   = "default"
	constLogAppending = "appending"
	constLogDisabled  = "disabled"
)

// enLogging defines possible option how to handle configuration.
type enLogging int

const (
	// Default represents saving into new file and backing up older with suffix
	Default enLogging = iota + 1

	// Appending represents saving all logs into same file
	Appending

	// Disabled represents disabled logging
	Disabled
)

// config defines structure of application configuration.
type config struct {
	daemonMode         bool
	defaultUser        string
	autologin          bool
	autologinSession   string
	minTty             int
	pamService         string
	switchTTY          bool
	printIssue         bool
	lang               string
	dbusLaunch         bool
	xinitrcLaunch      bool
	verticalSelection  bool
	logging            enLogging
	xorgArgs           string
	loggingFile        string
	dynamicMotd        bool
	fgColor            string
	bgColor            string
	displayStartScript string
	displayStopScript  string
	sessionWrapper     string
}

func newDefaultConfig() *config {
	c := &config{
		daemonMode:         false,
		minTty:             1,
		switchTTY:          true,
		printIssue:         true,
		defaultUser:        "",
		autologin:          false,
		autologinSession:   "",
		lang:               "en_US.UTF-8",
		dbusLaunch:         true,
		xinitrcLaunch:      false,
		verticalSelection:  false,
		logging:            Default,
		xorgArgs:           "",
		loggingFile:        "",
		dynamicMotd:        false,
		fgColor:            "",
		bgColor:            "",
		displayStartScript: "",
		displayStopScript:  "",
		sessionWrapper:     "",
		pamService:         "emptty",
	}

	tmpLang, ok := os.LookupEnv("LANG")
	if ok && tmpLang != "" {
		c.lang = tmpLang
	}
	return c
}

// LoadConfig handles loading of application configuration.
func loadConfig(c *config, path string) error {
	file, err := ini.Load(path)
	if err != nil {
		return err
	}
	empttySession, err := file.GetSection("emptty")
	if err != nil {
		return err
	}

	log.Println("load:", path)

	setIntValue(&c.minTty, empttySession, confMinTTY)
	setStringValue(&c.pamService, empttySession, "PAM_SERVICE")
	setBoolValue(&c.switchTTY, empttySession, "SWITCH_TTY")
	setBoolValue(&c.printIssue, empttySession, "PRINT_ISSUE")
	setStringValue(&c.defaultUser, empttySession, "DEFAULT_USER")
	setBoolValue(&c.autologin, empttySession, "AUTOLOGIN")
	setStringValue(&c.autologinSession, empttySession, "AUTOLOGIN")
	setStringValue(&c.lang, empttySession, "LANG")
	setBoolValue(&c.dbusLaunch, empttySession, "DBUS_LAUNCH")
	setBoolValue(&c.xinitrcLaunch, empttySession, "XINITRC_LAUNCH")
	setBoolValue(&c.verticalSelection, empttySession, "VERTICAL_SELECTION")
	varString := ""
	setStringValue(&varString, empttySession, "LOGGING")
	switch varString {
	case constLogDisabled:
		c.logging = Disabled
	case constLogAppending:
		c.logging = Appending
	case constLogDefault:
		c.logging = Default
	}

	setStringValue(&c.xorgArgs, empttySession, "XORG_ARGS")
	setStringValue(&c.loggingFile, empttySession, "LOGGING_FILE")
	setBoolValue(&c.dynamicMotd, empttySession, "DYNAMIC_MOTD")
	setStringValue(&c.fgColor, empttySession, "FG_COLOR")
	setStringValue(&c.bgColor, empttySession, "BG_COLOR")
	setStringValue(&c.displayStartScript, empttySession, "DISPLAY_START_SCRIPT")
	setStringValue(&c.displayStopScript, empttySession, "DISPLAY_STOP_SCRIPT")
	setStringValue(&c.sessionWrapper, empttySession, "SESSION_WRAPPER")
	return nil
}

func setBoolValue(vari *bool, s *ini.Section, key string) error {
	k, err := s.GetKey(key)
	if err != nil || k == nil {
		return nil
	}

	val, err := k.Bool()
	if err != nil {
		return nil
	}
	*(vari) = val

	return nil
}

func setIntValue(vari *int, s *ini.Section, key string) error {
	log.Println(key)
	k, err := s.GetKey(key)
	if err != nil || k == nil {
		return nil
	}

	val, err := k.Int()
	if err != nil {
		return nil
	}
	*(vari) = val

	return nil
}

func setStringValue(vari *string, s *ini.Section, key string) error {
	k, err := s.GetKey(key)
	if err != nil && k == nil {
		return nil
	}
	*(vari) = k.String()

	return nil
}

// emptty.d/00_xxx.conf
func loadConfigDir(c *config, dir string) error {
	filepath.WalkDir(pathConfigDir, func(path string, d fs.DirEntry, err error) error {
		if pathConfigDir == path {
			return nil
		}

		if d.IsDir() {
			return fs.SkipDir
		}

		baseName := filepath.Base(path)

		reg, err1 := regexp.Compile(`^\d\d_[\S]*.conf$`)
		if err1 != nil {
			return nil
		}
		if !reg.MatchString(baseName) {
			return nil
		}

		loadConfig(c, path)
		return nil
	})
	return nil
}

// Parse TTY number.
func parseTTY(tty string, defaultValue string) int {
	val, err := strconv.ParseInt(sanitizeValue(tty, defaultValue), 10, 32)
	if err != nil {
		return 0
	}
	return int(val)
}

// Parse logging option
func parseLogging(strLogging string, defaultValue string) enLogging {
	val := sanitizeValue(strLogging, defaultValue)
	switch val {
	case constLogDisabled:
		return Disabled
	case constLogAppending:
		return Appending
	case constLogDefault:
		return Default
	}
	return Default
}

// Returns TTY number converted to string
func (c *config) strTTY() string {
	return strconv.Itoa(c.minTty)
}
