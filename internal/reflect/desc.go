/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package reflect

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"

	"github.com/cloudwego/frugal/internal/defs"
)

var (
	sdsmu sync.Mutex
	sds   = newMapStructDesc()
)

func getOrcreateStructDesc(rv reflect.Value) (*structDesc, error) {
	sd := sds.Get(rvTypePtr(rv))
	if sd != nil {
		return sd, nil
	}
	return createStructDesc(rv)
}

func getStructDesc(rv reflect.Value) *structDesc {
	return sds.Get(rvTypePtr(rv))
}

var errType = errors.New("not pointer to struct")

func createStructDesc(rv reflect.Value) (*structDesc, error) {
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		if rt.Kind() != reflect.Ptr {
			return nil, errType
		}
		rt = rt.Elem()
		if rt.Kind() != reflect.Struct {
			return nil, errType
		}
	}
	abiType := rtTypePtr(rt)
	sdsmu.Lock()
	defer sdsmu.Unlock()
	if sd := sds.Get(abiType); sd != nil {
		return sd, nil
	}
	sd, err := newStructDescAndPrefetch(rt)
	if err != nil {
		return nil, err
	}
	sds.Set(abiType, sd)
	if rv.Kind() == reflect.Ptr {
		sds.Set(rvTypePtr(rv), sd) // *struct and struct share the same structDesc
	}
	return sd, nil
}

var prefetchStructDescCache = map[reflect.Type]*structDesc{}

func newStructDescAndPrefetch(t reflect.Type) (*structDesc, error) {
	if sd := prefetchStructDescCache[t]; sd != nil {
		return sd, nil
	}
	sd, err := newStructDesc(t)
	if err != nil {
		return nil, err
	}
	prefetchStructDescCache[t] = sd
	if err := prefetchSubStructDesc(sd); err != nil {
		delete(prefetchStructDescCache, t)
		return nil, err
	}
	return sd, nil
}

func prefetchSubStructDesc(d *structDesc) error {
	for i := range d.fields {
		f := d.fields[i]
		switch f.Type.T {
		case tSTRUCT, tMAP, tLIST, tSET:
			if err := fetchStructDesc(f.Type); err != nil {
				return err
			}
		}
	}
	return nil
}

func fetchStructDesc(t *tType) error {
	if t.T == tMAP {
		err := fetchStructDesc(t.K)
		if err != nil {
			return err
		}
		return fetchStructDesc(t.V)
	}
	if t.T == tLIST || t.T == tSET {
		return fetchStructDesc(t.V)
	}
	if t.T != tSTRUCT || t.Sd != nil {
		return nil
	}
	sd, err := newStructDescAndPrefetch(t.RT)
	if err != nil {
		return err
	}
	t.Sd = sd
	return nil
}

type iInitDefault interface {
	InitDefault()
}

type structDesc struct {
	rt reflect.Type // always Kind() == reflect.Struct

	// tmp var for direct type, need to copy to heap before using unsafe.Pointer
	rvPool sync.Pool

	maxID    uint16 // protect fieldIdx
	fieldIdx []int  // directly maps field id to Field for performance
	fields   []*tField

	hasInitFunc bool         // true if reflect.Type implements iInitDefault
	initFunc    iInitDefault // need to change the data pointer when calling

	hasUnknownFields    bool // for the _unknownFields feature
	unknownFieldsOffset uintptr

	fixedLenFieldSize int   // sum of f.EncodedSize() > 0
	varLenFields      []int // maps to fields. list of fields that f.EncodedSize() <= 0
	requiredFieldIDs  []uint16
}

func newStructDesc(t reflect.Type) (*structDesc, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errType
	}
	ff, err := defs.DoResolveFields(t)
	if err != nil {
		return nil, err
	}
	d := &structDesc{rt: t}
	d.rvPool = sync.Pool{
		New: func() interface{} {
			rv := reflect.New(t)
			return &rv
		},
	}
	d.fromDefsFields(ff)

	eface := reflect.New(t).Interface()
	d.initFunc, d.hasInitFunc = eface.(iInitDefault)

	f, ok := t.FieldByName("_unknownFields")
	if ok && f.Type.Kind() == reflect.Slice && f.Type.Elem().Kind() == reflect.Uint8 {
		d.hasUnknownFields = true
		d.unknownFieldsOffset = f.Offset
	}
	return d, nil
}

func (d *structDesc) Name() string {
	return d.rt.String()
}

func (d *structDesc) GetField(fid uint16) *tField {
	if fid > d.maxID {
		return nil
	}
	i := d.fieldIdx[fid]
	if i < 0 {
		return nil
	}
	return d.fields[i]
}

func (d *structDesc) fromDefsFields(ff []defs.Field) {
	maxFieldID := uint16(0)
	for _, f := range ff {
		if f.ID > maxFieldID {
			maxFieldID = f.ID
		}
	}
	d.maxID = maxFieldID
	d.fieldIdx = make([]int, maxFieldID+1)
	for i := range d.fieldIdx {
		d.fieldIdx[i] = -1
	}
	fields := make([]tField, len(ff))
	d.fields = make([]*tField, len(ff))
	for i, f := range ff {
		d.fields[i] = &fields[i]
		d.fields[i].fromDefsField(f)
		d.fieldIdx[f.ID] = i
	}
	d.varLenFields = make([]int, 0, len(ff))
	d.requiredFieldIDs = make([]uint16, 0, len(ff))
	for i, f := range d.fields {
		if n := f.EncodedSize(); n > 0 {
			d.fixedLenFieldSize += n
		} else {
			d.varLenFields = append(d.varLenFields, i)
		}
		if f.Spec == defs.Required {
			d.requiredFieldIDs = append(d.requiredFieldIDs, f.ID)
		}
	}
}

type tField struct {
	ID     uint16
	Offset uintptr
	Type   *tType

	Spec    defs.Requiredness
	Default unsafe.Pointer

	NoCopy             bool
	CanSkipEncodeIfNil bool
	CanSkipIfDefault   bool
}

var containerTypes = [256]bool{
	tMAP:  true,
	tLIST: true,
	tSET:  true,
}

var typeToSize = [256]int8{
	tBOOL:   1,
	tBYTE:   1,
	tDOUBLE: 8,
	tI16:    2,
	tI32:    4,
	tI64:    8,
	tENUM:   4,
}

// EncodedSize returns encoded size of the field, -1 if can not be determined.
func (f *tField) EncodedSize() int {
	if f.Type.IsPointer { // may be nil, then skip encoding
		return -1
	}
	if f.Spec == defs.Optional {
		return -1 // may have default value, then skip encoding
	}
	if f.Type.FixedSize > 0 {
		return fieldHeaderLen + f.Type.FixedSize // type + id + len
	}
	return -1
}

func (f *tField) fromDefsField(x defs.Field) {
	f.ID = x.ID
	f.Offset = uintptr(x.F)
	f.Type = newTType(x.Type)
	f.Spec = x.Spec

	t := f.Type

	f.NoCopy = (x.Opts & defs.NoCopy) != 0
	if f.NoCopy && f.Type.WT != tSTRING {
		// never goes here, defs will check the tag
		panic("[BUG] nocopy on non-STRING type")
	}
	// for map or slice, t.IsPointer() is false,
	// but we can consider the types as pointer as per lang spec
	// for defs.T_binary, actually it's []byte, like tLIST
	f.CanSkipEncodeIfNil = f.Spec == defs.Optional &&
		(t.Tag == defs.T_pointer || t.Tag == defs.T_binary || containerTypes[t.T])

	// for SkipEncodeDefault
	v := x.Default
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		return
	}
	f.Default = unsafe.Pointer(v.UnsafeAddr())
	f.CanSkipIfDefault = (f.Spec == defs.Optional) &&
		t.Tag != defs.T_pointer && // normally if fields with default values, it's non-pointer
		f.Default != nil // the field must have default value
}
