PREFIX = /usr

export GO111MODULE=on
GOBUILD = go build -v

all: build

build:
	${GOBUILD} -o emptty

install:
	install -Dm755 emptty ${DESTDIR}/usr/bin/emptty
	mkdir -p ${DESTDIR}/etc/emptty
	install -Dm644 res/conf ${DESTDIR}/etc/emptty/conf
	install -Dm644 res/pam-debian ${DESTDIR}/etc/pam.d/emptty
	install -Dm644 res/systemd-service ${DESTDIR}/usr/lib/systemd/system/emptty.service

clean:
	rm -f emptty

