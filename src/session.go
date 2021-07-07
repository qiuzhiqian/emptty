package src

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Session interface {
	Start() error
}

type BaseSession struct {
	usr  *sysuser
	d    *desktop
	conf *config
}

type XSession struct {
	BaseSession
}

func (s *XSession) Start() error {
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
	xorg.Env = append(os.Environ())
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

type WaylandSession struct {
	BaseSession
}

func (s *WaylandSession) Start() error {
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

func NewSession(usr *sysuser, d *desktop, conf *config) Session {
	switch d.env {
	case Wayland:
		return &WaylandSession{
			BaseSession{
				usr:  usr,
				d:    d,
				conf: conf,
			},
		}
	case Xorg:
		return &XSession{
			BaseSession{
				usr:  usr,
				d:    d,
				conf: conf,
			},
		}
	}
	return nil
}
