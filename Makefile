PREFIX = /usr

export GO111MODULE=on
GOBUILD = go build -v

all: build

build:
	${GOBUILD} -o emptty

install:
	install -Dm755 emptty ${DESTDIR}/usr/bin/emptty
	install -Dm644 res/emptty.conf ${DESTDIR}/etc/emptty.conf
	install -Dm644 res/pam-debian ${DESTDIR}/etc/pam.d/emptty
	install -Dm644 res/systemd-service ${DESTDIR}/usr/lib/systemd/system/emptty.service

	mkdir -pv ${DESTDIR}${PREFIX}/share/dbus-1/system.d
	install -Dm644 res/com.github.qiuzhiqian.Emptty.conf ${DESTDIR}${PREFIX}/share/dbus-1/system.d/com.github.qiuzhiqian.Emptty.conf

clean:
	rm -f emptty

