package src

import (
	"fmt"
	"log"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const intro = `
<node>
	<interface name="com.github.qiuzhiqian.Emptty">
		<method name="SwitchUser">
			<arg direction="in" type="s"/>
		</method>
	</interface>` + introspect.IntrospectDataString + `</node> `

type Manager struct {
}

func (m *Manager) SwitchUser(user string) *dbus.Error {
	log.Println(user)
	return nil
}

func Server() {
	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	f := &Manager{}
	conn.Export(f, "/com/github/qiuzhiqian/Emptty", "com.github.qiuzhiqian.Emptty")
	conn.Export(introspect.Introspectable(intro), "/com/github/qiuzhiqian/Emptty",
		"org.freedesktop.DBus.Introspectable")

	reply, err := conn.RequestName("com.github.qiuzhiqian.Emptty",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		//panic(err)
		log.Println("has error")
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken")
		os.Exit(1)
	}
	log.Println("Listening on com.github.qiuzhiqian.Emptty / /com/github/qiuzhiqian/Emptty ...")
	select {}
}

/*func main() {
	Server()
}*/
