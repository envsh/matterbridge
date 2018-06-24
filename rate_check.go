package btox

import (
	"log"
	"time"

	"github.com/juju/ratelimit"
	"github.com/kitech/godsts/lists/arraylist"
	"github.com/kitech/godsts/maps/hashmap"
)

type RateCheck2 struct {
	topics *hashmap.Map // peerPubkey@groupTitle => *ratelimit.Bucket
	banlst *hashmap.Map // peerPubkey@groupTitle => *ratelimit.Bucket
}

func NewRateCheck2() *RateCheck2 {
	this := &RateCheck2{}
	this.topics = hashmap.New()
	this.banlst = hashmap.New()
	return this
}

func (this *RateCheck2) TakeAvalible(topic string) bool {

	if bktx, found := this.banlst.Get(topic); found {
		log.Println("baning:", topic, bktx == nil)
		return false
	}

	// avalible here
	if !this.topics.Has(topic) {
		this.topics.Put(topic, ratelimit.NewBucket(6*time.Second/time.Duration(4), 4))
	}

	bktx, _ := this.topics.Get(topic)
	bkt := bktx.(*ratelimit.Bucket)
	freen := bkt.TakeAvailable(1)
	log.Println("freen:", freen)
	if freen == 0 {
		log.Println("baned:", topic)
		tmer := time.AfterFunc(30*time.Minute, func() { this.banlst.Remove(topic) })
		this.banlst.Put(topic, tmer)
		return false
	}
	return true
}

///
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
