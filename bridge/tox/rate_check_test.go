package btox

import (
	"log"
	"testing"
	"time"
)

func TestRChk0(t *testing.T) {
	for i := 0; i < 50; i++ {
		log.Println(grc.TryPut("hehehhe"))
		time.Sleep(1 * time.Second)
	}
}
