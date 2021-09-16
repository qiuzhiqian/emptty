package src

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Daemon struct {
	sessions []*BaseSession
}

func NewDaemon() *Daemon {
	return &Daemon{
		sessions: make([]*BaseSession, 0),
	}
}

const (
	envXdgConfigHome   = "XDG_CONFIG_HOME"
	envXdgRuntimeDir   = "XDG_RUNTIME_DIR"
	envXdgSessionId    = "XDG_SESSION_ID"
	envXdgSessionType  = "XDG_SESSION_TYPE"
	envXdgSessionClass = "XDG_SESSION_CLASS"
	envXdgSeat         = "XDG_SEAT"
	envHome            = "HOME"
	envPwd             = "PWD"
	envUser            = "USER"
	envLogname         = "LOGNAME"
	envXauthority      = "XAUTHORITY"
	envDisplay         = "DISPLAY"
	envShell           = "SHELL"
	envLang            = "LANG"
	envPath            = "PATH"
	envDesktopSession  = "DESKTOP_SESSION"
	envXdgSessDesktop  = "XDG_SESSION_DESKTOP"
	envSessionBusAddr  = "DBUS_SESSION_BUS_ADDRESS"
)

// Login into graphical environment
func login(conf *config) {
	daemon := NewDaemon()
	go Server()

	var wg sync.WaitGroup

	if len(daemon.sessions) == 0 {
		session := NewSession(conf)
		daemon.sessions = append(daemon.sessions, session)
		wg.Add(1)
		go session.Start(&wg)
	}

	wg.Wait()
}

// Prepares environment and env variables for authorized user.
func defineEnvironment(usr *sysuser, conf *config, d *desktop) {
	defineSpecificEnvVariables(usr)

	usr.setenv(envHome, usr.homedir)
	usr.setenv(envPwd, usr.homedir)
	usr.setenv(envUser, usr.username)
	usr.setenv(envLogname, usr.username)
	usr.setenv(envXdgConfigHome, usr.homedir+"/.config")
	usr.setenv(envXdgRuntimeDir, "/run/user/"+usr.strUid())
	usr.setenv(envXdgSeat, "seat0")
	usr.setenv(envXdgSessionClass, "user")
	shell := getUserShell(usr)
	if shell == "" {
		shell = "/usr/bin/bash"
	}
	usr.setenv(envShell, shell)
	usr.setenv(envLang, conf.lang)
	usr.setenv(envPath, os.Getenv(envPath))

	if d.name != "" {
		usr.setenv(envDesktopSession, d.name)
		usr.setenv(envXdgSessDesktop, d.name)
	} else if d.child != nil && d.child.name != "" {
		usr.setenv(envDesktopSession, d.child.name)
		usr.setenv(envXdgSessDesktop, d.child.name)
	}

	dbusAddr, ok := os.LookupEnv(envSessionBusAddr)
	log.Println("dbus env:", dbusAddr, ok)
	if ok && dbusAddr != "" {
		usr.setenv(envSessionBusAddr, dbusAddr)
	}

	log.Print("Defined Environment")

	// create XDG folder
	if !fileExists(usr.getenv(envXdgRuntimeDir)) {
		err := os.MkdirAll(usr.getenv(envXdgRuntimeDir), 0700)
		handleErr(err)

		// Set owner of XDG folder
		os.Chown(usr.getenv(envXdgRuntimeDir), usr.uid, usr.gid)

		log.Print("Created XDG folder")
	} else {
		log.Print("XDG folder already exists, no need to create")
	}

	os.Chdir(usr.getenv(envPwd))
}

// Reads default shell of authorized user.
func getUserShell(usr *sysuser) string {
	out, err := exec.Command("/usr/bin/getent", "passwd", usr.strUid()).Output()
	handleErr(err)

	ent := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	shellCmdline := ent[6]
	info, err := os.Stat(shellCmdline)
	if err != nil || info.IsDir() {
		return ""
	}

	return shellCmdline
}

// Prepares command for starting GUI.
func prepareGuiCommand(usr *sysuser, d *desktop, conf *config) (*exec.Cmd, error) {
	strExec, allowStartupPrefix := getStrExec(d)
	shell := getUserShell(usr)
	if shell == "" {
		bashPath, err := exec.LookPath("bash")
		if err != nil {
			return nil, err
		}
		shell = bashPath
	}

	startScript := false

	if d.selection && d.child != nil {
		strExec = d.path + " " + d.child.exec
	} else {
		if d.env == Xorg && conf.xinitrcLaunch && allowStartupPrefix && !strings.Contains(strExec, ".xinitrc") && fileExists(usr.homedir+"/.xinitrc") {
			startScript = true
			allowStartupPrefix = false
			strExec = usr.homedir + "/.xinitrc " + strExec
		}

		// if has DBUS_SESSION_BUS_ADDRESS,need not run dbus-launch
		dbusAddr, ok := os.LookupEnv(envSessionBusAddr)
		if (!ok || dbusAddr == "") && !strings.Contains(strExec, "dbus-launch") && allowStartupPrefix {
			//strExec = "dbus-launch " + strExec
		}

		// check session wrapper
		if conf.sessionWrapper != "" {
			strExec = fmt.Sprintf("%s %s", conf.sessionWrapper, strExec)
		}
	}

	arrExec := strings.Split(strExec, " ")

	//need with sh --login
	var cmd *exec.Cmd
	if len(arrExec) > 1 {
		if startScript {
			cmd = cmdAsUser(usr, shell, "--login", "-c", strExec)
		} else {
			cmd = cmdAsUser(usr, shell, "--login", "-c", strExec)
		}
	} else {
		cmd = cmdAsUser(usr, shell, "--login", "-c", strExec)
	}

	return cmd, nil
}

// Gets exec path from desktop and returns true, if command allows dbus-launch.
func getStrExec(d *desktop) (string, bool) {
	if d.exec != "" {
		return d.exec, true
	}
	return d.path, false
}

// Finds free display for spawning Xorg instance.
func getFreeXDisplay() int {
	for i := 0; i < 32; i++ {
		filename := fmt.Sprintf("/tmp/.X%d-lock", i)
		if !fileExists(filename) {
			return i
		}
	}
	return 0
}

// Registers interrupt handler, that interrupts all mentioned Cmds.
func registerInterruptHandler(cmds ...*exec.Cmd) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGTERM)
	go handleInterrupt(c, cmds...)
}

// Catch interrupt signal chan and interrupts all mentioned Cmds.
func handleInterrupt(c chan os.Signal, cmds ...*exec.Cmd) {
	sig := <-c
	log.Println("Catched interrupt signal", sig)
	for _, cmd := range cmds {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}
}

// Runs display script, if defined
func runDisplayScript(scriptPath string) {
	if scriptPath != "" {
		if fileIsExecutable(scriptPath) {
			err := exec.Command(scriptPath).Run()
			if err != nil {
				log.Print(err)
			}
		} else {
			log.Print(scriptPath + " is not executable.")
		}
	}
}
