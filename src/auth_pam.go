//go:build !nopam
// +build !nopam

package src

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/msteinert/pam"
)

const tagPam = ""

var trans *pam.Transaction

// Handle PAM authentication of user.
// If user is successfully authorized, it returns sysuser.
//
// If autologin is enabled, it behaves as user has been authorized.
func authUser(conf *config) *sysuser {
	log.Println("authUser")
	var err error

	trans, err = pam.StartFunc(conf.pamService, conf.defaultUser, func(s pam.Style, msg string) (string, error) {
		log.Println("StartFunc", s, msg)
		switch s {
		case pam.PromptEchoOff:
			if conf.autologin {
				break
			}
			if conf.defaultUser != "" {
				hostname, _ := os.Hostname()
				fmt.Printf("%s login: %s\n", hostname, conf.defaultUser)
			}
			fmt.Print("Password: ")
			return readPassword()
		case pam.PromptEchoOn:
			if conf.autologin {
				break
			}
			log.Println("PromptEchoOn for login")
			hostname, _ := os.Hostname()
			fmt.Printf("%s login: ", hostname)
			log.Println("wait for login")
			input, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return "", err
			}
			return input[:len(input)-1], nil
		case pam.ErrorMsg:
			log.Print(msg)
			return "", nil
		case pam.TextInfo:
			log.Println("TextInfo for msg", msg)
			fmt.Println(msg)
			return "", nil
		}
		return "", errors.New("unrecognized message style")
	})
	if err != nil {
		return nil
	}

	err = trans.Authenticate(pam.Silent)
	if err != nil {
		bkpErr := errors.New(err.Error())
		username, _ := trans.GetItem(pam.User)
		addBtmpEntry(username, os.Getpid(), conf.strTTY())
		handleErr(bkpErr)
	}
	log.Print("Authenticate OK")

	/* Check account is valid */
	err = trans.AcctMgmt(pam.Silent)
	if err != nil {
		trans.ChangeAuthTok(pam.Silent)
		handleErr(err)
	}
	//handleErr(err)

	err = trans.SetItem(pam.Tty, "tty"+conf.strTTY())
	handleErr(err)

	tmpUsr, _ := trans.GetItem(pam.User)
	log.Println("tmp user:", tmpUsr)

	err = trans.OpenSession(pam.Silent)
	handleErr(err)

	pamUsr, _ := trans.GetItem(pam.User)
	usr, _ := user.Lookup(pamUsr)

	return getSysuser(usr)
}

// Handles close of PAM authentication
func closeAuth() {
	if trans != nil {
		err := trans.CloseSession(pam.Silent)
		trans = nil
		if err != nil {
			log.Println(err)
		}
	}
}

// Defines specific environmental variables defined by PAM
func defineSpecificEnvVariables(usr *sysuser) {
	if trans != nil {
		envs, _ := trans.GetEnvList()
		for key, value := range envs {
			usr.setenv(key, value)
		}
	}
}
