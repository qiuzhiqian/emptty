package src

import (
	"os"
	"strconv"

	ini "gopkg.in/ini.v1"
)

const (
	confTTYnumber          = "TTY_NUMBER"
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

	pathConfigFile = "/etc/emptty/conf"

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
	tty                int
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
}

// LoadConfig handles loading of application configuration.
func loadConfig(path string) (*config, error) {
	c := config{
		daemonMode:         false,
		tty:                0,
		switchTTY:          true,
		printIssue:         true,
		defaultUser:        "",
		autologin:          false,
		autologinSession:   "",
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
	}

	file, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	empttySession, err := file.GetSection("emptty")
	if err != nil {
		return nil, err
	}

	defLang := "en_US.UTF-8"
	tmpLang, ok := os.LookupEnv("LANG")
	if ok && tmpLang != "" {
		defLang = tmpLang
	}

	c.tty = empttySession.Key("TTY_NUMBER").MustInt(1)
	c.pamService = empttySession.Key("PAM_SERVICE").MustString("emptty")
	c.switchTTY = empttySession.Key("SWITCH_TTY").MustBool(true)
	c.printIssue = empttySession.Key("PRINT_ISSUE").MustBool(true)
	c.defaultUser = empttySession.Key("DEFAULT_USER").MustString("")
	c.autologin = empttySession.Key("AUTOLOGIN").MustBool(false)
	c.autologinSession = empttySession.Key("AUTOLOGIN_SESSION").MustString("")
	c.lang = empttySession.Key("LANG").MustString(defLang)
	c.dbusLaunch = empttySession.Key("DBUS_LAUNCH").MustBool(true)
	c.xinitrcLaunch = empttySession.Key("XINITRC_LAUNCH").MustBool(false)
	c.verticalSelection = empttySession.Key("VERTICAL_SELECTION").MustBool(false)
	val := empttySession.Key("LOGGING").MustString("default")
	switch val {
	case constLogDisabled:
		c.logging = Disabled
	case constLogAppending:
		c.logging = Appending
	case constLogDefault:
		c.logging = Default
	}

	c.xorgArgs = empttySession.Key("XORG_ARGS").MustString("")
	c.loggingFile = empttySession.Key("LOGGING_FILE").MustString("")
	c.dynamicMotd = empttySession.Key("DYNAMIC_MOTD").MustBool(false)
	c.fgColor = convertColor(empttySession.Key("FG_COLOR").MustString(""), true)
	c.bgColor = convertColor(empttySession.Key("BG_COLOR").MustString(""), false)
	c.displayStartScript = empttySession.Key("DISPLAY_START_SCRIPT").MustString("")
	c.displayStopScript = empttySession.Key("DISPLAY_STOP_SCRIPT").MustString("")

	return &c, nil
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
	return strconv.Itoa(c.tty)
}
