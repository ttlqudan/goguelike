// Copyright 2014,2015,2016,2017,2018,2019,2020 SeukWon Kang (kasworld@gmail.com)
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// positioned object managment in 2d space
package uuidposman

import (
	"fmt"
	"sync"

	"github.com/kasworld/findnear"
	"github.com/kasworld/wrapper"
)

type UUIDPosI interface {
	GetUUID() string
}

type UUIDPosIList []UUIDPosI

type DoFn func(fo UUIDPosI) bool

func (fo *UUIDPosMan) String() string {
	return fmt.Sprintf("UUIDPosMan[(%v %v) %v]",
		fo.XWrapper.GetWidth(), fo.YWrapper.GetWidth(),
		fo.Count())
}

type UUIDPosMan struct {
	mutex    sync.RWMutex        `prettystring:"hide"`
	uuid2obj map[string]UUIDPosI `prettystring:"simple"`
	uuid2pos map[string][2]int   `prettystring:"simple"`
	pos2objs [][]UUIDPosIList    `prettystring:"simple"`
	XWrapper *wrapper.Wrapper    `prettystring:"simple"`
	YWrapper *wrapper.Wrapper    `prettystring:"simple"`
	XWrap    func(i int) int
	YWrap    func(i int) int
}

func New(x, y int) *UUIDPosMan {
	rtn := UUIDPosMan{
		uuid2obj: make(map[string]UUIDPosI),
		uuid2pos: make(map[string][2]int),
		pos2objs: make([][]UUIDPosIList, x),
		XWrapper: wrapper.New(x),
		YWrapper: wrapper.New(y),
	}
	rtn.XWrap = rtn.XWrapper.GetWrapFn()
	rtn.YWrap = rtn.YWrapper.GetWrapFn()
	for i, _ := range rtn.pos2objs {
		rtn.pos2objs[i] = make([]UUIDPosIList, y)
	}
	return &rtn
}

func (fo *UUIDPosMan) Cleanup() {
	fo.mutex.Lock()
	defer fo.mutex.Unlock()
	for id, o := range fo.uuid2obj {
		delete(fo.uuid2obj, id)
		delete(fo.uuid2pos, id)
		pos, exist := fo.uuid2pos[id]
		if exist {
			fo.delObjAt(o, pos[0], pos[1])
		}
	}
}

func (fo *UUIDPosMan) Count() int {
	return len(fo.uuid2obj)
}

func (fo *UUIDPosMan) Wrap(x, y int) (int, int) {
	return fo.XWrap(x), fo.YWrap(y)
}

func (fo *UUIDPosMan) GetAllList() UUIDPosIList {
	rtn := make(UUIDPosIList, 0, len(fo.uuid2obj))
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()
	for _, v := range fo.uuid2obj {
		rtn = append(rtn, v)
	}
	return rtn
}

func (fo *UUIDPosMan) GetByUUID(id string) UUIDPosI {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()
	return fo.uuid2obj[id]
}

func (fo *UUIDPosMan) GetByXYAndUUID(id string, x, y int) (UUIDPosI, error) {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()

	x, y = fo.Wrap(x, y)

	obj, exist := fo.uuid2obj[id]
	if !exist {
		return nil, fmt.Errorf("obj not exist %v", id)
	}
	pos, exist := fo.uuid2pos[id]
	if !exist {
		return nil, fmt.Errorf("obj not exist %v", id)
	}
	if x != pos[0] || y != pos[1] {
		return obj, fmt.Errorf("pos not match %v %v %v", pos, x, y)
	}
	return obj, nil
}

func (fo *UUIDPosMan) GetXYByUUID(id string) (int, int, bool) {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()
	pos, exist := fo.uuid2pos[id]
	return pos[0], pos[1], exist
}

func (fo *UUIDPosMan) Get1stObjAt(x, y int) UUIDPosI {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()

	x, y = fo.Wrap(x, y)
	for _, v := range fo.pos2objs[x][y] {
		if v == nil {
			continue
		}
		return v
	}
	return nil
}

func (fo *UUIDPosMan) GetObjListAt(x, y int) UUIDPosIList {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()
	rtn := make(UUIDPosIList, 0, len(fo.pos2objs[x][y]))
	x, y = fo.Wrap(x, y)
	for _, v := range fo.pos2objs[x][y] {
		if v == nil {
			continue
		}
		rtn = append(rtn, v)
	}
	return rtn
}

func (fo *UUIDPosMan) Search1stByXYLenList(
	xylenlist findnear.XYLenList,
	sx, sy int,
	filterfn func(o UUIDPosI, x, y int, xylen findnear.XYLen) bool) (UUIDPosI, int, int) {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()

	for _, v := range xylenlist {
		x, y := fo.Wrap(sx+v.X, sy+v.Y)
		for _, o := range fo.pos2objs[x][y] {
			if o == nil {
				continue
			}
			if filterfn(o, x, y, v) {
				return o, x, y
			}
		}
	}
	return nil, 0, 0
}

// VPIXYObj use for viewport by xylenlist,
type VPIXYObj struct {
	I    int // xylen pos
	X, Y int // pos in uuidposman
	O    UUIDPosI
}

func (fo *UUIDPosMan) GetVPIXYObjByXYLenList(
	xylenlist findnear.XYLenList,
	sx, sy int, l int) []VPIXYObj {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()

	rtn := make([]VPIXYObj, 0)
	for i, v := range xylenlist {
		x, y := fo.Wrap(sx+v.X, sy+v.Y)
		for _, o := range fo.pos2objs[x][y] {
			if o == nil {
				continue
			}
			rtn = append(rtn, VPIXYObj{
				I: i,
				X: x,
				Y: y,
				O: o,
			})
			if len(rtn) >= l {
				return rtn
			}
		}
	}
	return rtn
}

func (fo *UUIDPosMan) IterByXYLenList(
	xylenlist findnear.XYLenList,
	sx, sy int, l int,
	stopFn func(o UUIDPosI, x, y int, i int, xylen findnear.XYLen) bool) {
	fo.mutex.RLock()
	defer fo.mutex.RUnlock()

loop:
	for i, v := range xylenlist {
		x, y := fo.Wrap(sx+v.X, sy+v.Y)
		for _, o := range fo.pos2objs[x][y] {
			if o == nil {
				continue
			}
			if stopFn(o, x, y, i, v) {
				break loop
			}
		}
	}
}

func (fo *UUIDPosMan) AddOrUpdateToXY(o UUIDPosI, x, y int) error {
	fo.mutex.Lock()
	defer fo.mutex.Unlock()

	if fo.uuid2obj[o.GetUUID()] != nil {
		if err := fo.delNolock(o); err != nil {
			return err
		}
	}
	return fo.addNolock(o, x, y)
}

func (fo *UUIDPosMan) AddToXY(o UUIDPosI, x, y int) error {
	fo.mutex.Lock()
	defer fo.mutex.Unlock()
	return fo.addNolock(o, x, y)
}
func (fo *UUIDPosMan) addNolock(o UUIDPosI, x, y int) error {
	id := o.GetUUID()
	if fo.uuid2obj[id] != nil {
		return fmt.Errorf("Fail to addNolock obj exist %v", o)
	}

	// x, y := o.GetXY()
	x, y = fo.Wrap(x, y)
	fo.uuid2obj[id] = o
	fo.uuid2pos[id] = [2]int{x, y}
	fo.addObj2Pos(o, x, y)
	return nil
}

func (fo *UUIDPosMan) Del(o UUIDPosI) error {
	fo.mutex.Lock()
	defer fo.mutex.Unlock()
	return fo.delNolock(o)
}
func (fo *UUIDPosMan) delNolock(o UUIDPosI) error {
	id := o.GetUUID()
	if fo.uuid2obj[id] == nil {
		return fmt.Errorf("Fail to delNolock obj not exist %v", o)
	}

	pos, exist := fo.uuid2pos[id]
	if !exist {
		return fmt.Errorf("Fail to delNolock obj not found %v", o)
	}
	if err := fo.delObjAt(o, pos[0], pos[1]); err == nil {
		delete(fo.uuid2obj, id)
		delete(fo.uuid2pos, id)
		return nil
	} else {
		return err
	}
}

func (fo *UUIDPosMan) UpdateToXY(o UUIDPosI, newx, newy int) error {
	fo.mutex.Lock()
	defer fo.mutex.Unlock()
	newx, newy = fo.Wrap(newx, newy)
	oldpos, exist := fo.uuid2pos[o.GetUUID()] //o.GetPos()
	if !exist {
		return fmt.Errorf("Fail to UpdateToXY obj not found %v", o)
	}
	if err := fo.delObjAt(o, oldpos[0], oldpos[1]); err == nil {
		fo.uuid2pos[o.GetUUID()] = [2]int{newx, newy}
		fo.addObj2Pos(o, newx, newy)
		return nil
	} else {
		return err
	}
}

///////////////////////////////////////////////////////////

func (fo *UUIDPosMan) addObj2Pos(o UUIDPosI, x, y int) {
	for i, v := range fo.pos2objs[x][y] {
		if v == nil {
			fo.pos2objs[x][y][i] = o
			return
		}
	}
	fo.pos2objs[x][y] = append(fo.pos2objs[x][y], o)
}

func (fo *UUIDPosMan) delObjAt(o UUIDPosI, x, y int) error {
	for i, v := range fo.pos2objs[x][y] {
		if v == nil {
			continue
		}
		if v.GetUUID() == o.GetUUID() {
			fo.pos2objs[x][y][i] = nil
			return nil
		}
	}
	return fmt.Errorf("Fail to delObjAt obj not at [%v %v] %v", x, y, o)
}
