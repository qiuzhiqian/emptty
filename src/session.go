package src

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type Session interface {
	Start() error
}

type BaseSession struct {
	usr  *sysuser
	d    *desktop
	conf *config
}

func (bs *BaseSession) auth() error {
	bs.usr = authUser(bs.conf)
	return nil
}

func (bs *BaseSession) desktopSelect() error {
	//var d *desktop
	var usrLang = ""
	bs.d, usrLang = loadUserDesktop(bs.usr.homedir)

	if bs.d == nil || (bs.d != nil && bs.d.selection) {
		selectedDesktop := selectDesktop(bs.usr, bs.conf)
		if bs.d != nil && bs.d.selection {
			bs.d.child = selectedDesktop
			bs.d.env = bs.d.child.env
		} else {
			bs.d = selectedDesktop
		}
	}

	if usrLang != "" {
		bs.conf.lang = usrLang
	}

	defineEnvironment(bs.usr, bs.conf, bs.d)
	return nil
}

func (s *BaseSession) startX() error {
	freeDisplay := strconv.Itoa(getFreeXDisplay())

	// Set environment
	s.usr.setenv(envXdgSessionType, "x11")
	s.usr.setenv(envXauthority, s.usr.getenv(envXdgRuntimeDir)+"/.emptty-xauth")
	s.usr.setenv(envDisplay, ":"+freeDisplay)
	os.Setenv(envXauthority, s.usr.getenv(envXauthority))
	os.Setenv(envDisplay, s.usr.getenv(envDisplay))
	log.Print("Defined Xorg environment")

	// create xauth
	os.Remove(s.usr.getenv(envXauthority))

	// generate mcookie
	cmd := cmdAsUser(s.usr, "/usr/bin/mcookie")
	mcookie, err := cmd.Output()
	handleErr(err)
	log.Print("Generated mcookie")

	// generate xauth
	cmd = cmdAsUser(s.usr, "/usr/bin/xauth", "add", s.usr.getenv(envDisplay), ".", string(mcookie))
	_, err = cmd.Output()
	handleErr(err)

	log.Print("Generated xauthority")

	// start X
	log.Print("Starting Xorg")

	xorgArgs := []string{"vt" + s.conf.strTTY(), s.usr.getenv(envDisplay)}

	if s.conf.xorgArgs != "" {
		arrXorgArgs := strings.Split(s.conf.xorgArgs, " ")
		xorgArgs = append(xorgArgs, arrXorgArgs...)
	}

	xorg := exec.Command("/usr/bin/Xorg", xorgArgs...)
	xorg.Env = append(xorg.Env, os.Environ()...)
	xorg.Start()
	if xorg.Process == nil {
		handleStrErr("Xorg is not running")
	}
	log.Print("Started Xorg")

	disp := &xdisplay{}
	disp.dispName = s.usr.getenv(envDisplay)
	handleErr(disp.openXDisplay())

	// make utmp entry
	utmpEntry := addUtmpEntry(s.usr.username, xorg.Process.Pid, s.conf.strTTY(), s.usr.getenv(envDisplay))
	log.Print("Added utmp entry")

	// start xinit
	xinit, err := prepareGuiCommand(s.usr, s.d, s.conf)
	if err != nil {
		panic(err)
	}
	registerInterruptHandler(xinit, xorg)
	log.Println("Starting ", xinit)
	err = xinit.Start()
	if err != nil {
		xorg.Process.Signal(os.Interrupt)
		xorg.Wait()
		handleErr(err)
	}

	xinit.Wait()
	log.Println(xinit, "finished")

	// Stop Xorg
	xorg.Process.Signal(os.Interrupt)
	xorg.Wait()
	log.Print("Interrupted Xorg")

	// Remove auth
	os.Remove(s.usr.getenv(envXauthority))
	log.Print("Cleaned up xauthority")

	// End utmp entry
	endUtmpEntry(utmpEntry)
	log.Print("Ended utmp entry")
	return nil
}

func (s *BaseSession) startWayland() error {
	// Set environment
	s.usr.setenv(envXdgSessionType, "wayland")
	log.Print("Defined Wayland environment")

	// start Wayland
	wayland, err := prepareGuiCommand(s.usr, s.d, s.conf)
	if err != nil {
		panic(err)
	}
	registerInterruptHandler(wayland)
	log.Println("Starting ", wayland)
	err = wayland.Start()
	handleErr(err)

	// make utmp entry
	utmpEntry := addUtmpEntry(s.usr.username, wayland.Process.Pid, s.conf.strTTY(), "")
	log.Print("Added utmp entry")

	wayland.Wait()
	log.Println(wayland, "finished")

	// end utmp entry
	endUtmpEntry(utmpEntry)
	log.Print("Ended utmp entry")
	return nil
}

func (bs *BaseSession) Start(wg *sync.WaitGroup) error {
	log.Println("begin auth")
	bs.auth()
	bs.desktopSelect()

	defineEnvironment(bs.usr, bs.conf, bs.d)
	runDisplayScript(bs.conf.displayStartScript)

	switch bs.d.env {
	case Xorg:
		bs.startX()
	case Wayland:
		bs.startWayland()
	}
	wg.Done()
	return nil
}

func NewSession(conf *config) *BaseSession {
	session := BaseSession{
		conf: conf,
	}

	return &session
}

func (bs *BaseSession) Destory() error {
	closeAuth()
	runDisplayScript(bs.conf.displayStopScript)
	return nil
}
