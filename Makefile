VERSION         :=      $(shell cat ./VERSION)

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.Entry=main"
GOVVV=`govvv -flags -version ${VERSION}|sed 's/=/=GOVVV-/g'`


all:
	go build -v -ldflags "${GOVVV}" .

