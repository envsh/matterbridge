package btox

import (
	"time"

	"github.com/kitech/godsts/lists/arraylist"
	"github.com/kitech/godsts/maps/hashmap"
)

var grc = NewRateCheck(6, 4)

type RateCheck struct {
	perUint int64        // seconds, a range
	rate    int          //
	topics  *hashmap.Map // key(string) => *arraylist.List
}

func NewRateCheck(perUint, rate int) *RateCheck {
	this := &RateCheck{}
	this.perUint = int64(perUint)
	this.rate = rate
	this.topics = hashmap.New()
	return this
}

// return true: ok, false: rate exceed
func (this *RateCheck) TryPut(topic string, extras ...string) bool {
	nowt := time.Now()

	var lst *arraylist.List
	if lstx, found := this.topics.Get(topic); found {
		lst = lstx.(*arraylist.List)
	} else {
		lst = arraylist.New()
		this.topics.Put(topic, lst)
	}

	lst2 := lst.Select(func(index int, value interface{}) bool {
		elmt := value.(time.Time)
		return nowt.Sub(elmt).Nanoseconds() < this.perUint*1000000000
	})
	this.topics.Put(topic, lst2)

	if lst2.Size() < this.rate {
		lst2.Add(nowt)
		return true
	}
	return false
}

// see also token bucket, leaked bucket algorithems.
