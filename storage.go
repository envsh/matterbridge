package btox

import (
	"fmt"
	"gopp"
	"time"

	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
)

// 是否要考虑JSONDB？
// 存储好友所在群组记录，某个好友在哪些群中
type Storage struct {
	dbe *xorm.Engine
}

func newStorage() *Storage {
	this := &Storage{}

	dsn := "toxbrg.sqlite3?cache=shared&mode=rwc"
	dbe, err := xorm.NewEngine("sqlite3", dsn)
	gopp.ErrPrint(err)
	dbe.ShowSQL(true)

	err = dbe.Ping()
	gopp.ErrPrint(err)

	this.dbe = dbe
	this.SetWAL(true)
	return this
}

func (this *Storage) SetWAL(enable bool) {
	_, err := this.dbe.Exec("PRAGMA journal_mode=WAL;")
	gopp.ErrPrint(err)
	_, err = this.dbe.Exec(fmt.Sprintf("PRAGMA journal_size_limit=%d;", 3*1000*1000)) // 3MB
	gopp.ErrPrint(err)
	// others: wal_checkpoint, wal_autocheckpoint, synchronous, cache_size
	_, err = this.dbe.Exec("PRAGMA locking_mode=EXCLUSIVE;")
	gopp.ErrPrint(err)
}

func (this *Storage) join(MemberId, RoomName string) (err error) {
	rec := RoomMember{}
	rec.MemberId = MemberId
	rec.RoomName = RoomName
	rec.Created = time.Now()
	rec.Updated = rec.Created

	id, err := this.dbe.Insert(&rec)
	gopp.ErrPrint(err, id)
	if err != nil {
		err = nil
		rec = RoomMember{Updated: time.Now(), Disabled: 0}
		n, err := this.dbe.Where("member_id=? and room_name=?", MemberId, RoomName).
			MustCols("Disabled").Update(&rec)
		gopp.ErrPrint(err, n)
	}

	return
}

func (this *Storage) leave(MemberId, RoomName string) (err error) {
	rec := RoomMember{}
	rec.Disabled = 1
	rec.Updated = time.Now()

	n, err := this.dbe.Where("member_id=? and room_name=?", MemberId, RoomName).Update(&rec)
	gopp.ErrPrint(err, n)
	return
}

func (this *Storage) getMembersByRoomName(RoomName string) (rets []RoomMember, err error) {

	err = this.dbe.Where("room_name=? and disabled=?", RoomName, 0).Find(&rets)
	gopp.ErrPrint(err)
	return
}

func (this *Storage) getRoomsByMemberId(MemberId string) (rets []RoomMember, err error) {
	err = this.dbe.Where("member_id=? and disabled=?", MemberId, 0).Find(&rets)
	gopp.ErrPrint(err)
	return
}

func (this *Storage) getAllRoomMembers() (rets []RoomMember, err error) {
	err = this.dbe.Where("1=1").Find(&rets)
	return
}

type RoomMember struct {
	MemberId string
	RoomName string
	Created  time.Time
	Disabled int
	Updated  time.Time
	Muted    int
}
