VERSION         :=      $(shell cat ./VERSION)

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.Entry=main"
GOVVV=`govvv -flags -version ${VERSION}|sed 's/=/=GOVVV-/g'`

export PKG_CONFIG_PATH=/opt/toxcore-static2/lib64/pkgconfig
all:
	# TODO hard code path, -race
	CGO_LDFLAGS="-L/opt/toxcore-static2/lib64 -lopus -lsodium" \
		go build -v -i -pkgdir ${HOME}/oss/pkg/linux_amd64 -ldflags "${GOVVV}" .

