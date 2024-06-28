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

	"github.com/cloudwego/frugal/internal/binary/defs"
)

var mapFieldDescWriteMu sync.Mutex
var fds = newMapFieldDesc()

func getOrcreateFieldDesc(rv reflect.Value) (*fieldDesc, error) {
	fd := fds.Get(rvTypePtr(rv))
	if fd != nil {
		return fd, nil
	}
	return createFieldDesc(rv)
}

func getFieldDesc(rv reflect.Value) *fieldDesc {
	return fds.Get(rvTypePtr(rv))
}

var errType = errors.New("not pointer to struct")

func createFieldDesc(rv reflect.Value) (*fieldDesc, error) {
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		if rt.Kind() != reflect.Ptr {
			return nil, errType
		}
		rt = rt.Elem()
		if rt.Kind() != reflect.Struct {
			return nil, errType
		}
		// XXX: only for rvTypePtr
		// make sure rvTypePtr returns reflect.Struct abiType
		rv = reflect.New(rt).Elem()
	}
	abiType := rvTypePtr(rv)
	mapFieldDescWriteMu.Lock()
	defer mapFieldDescWriteMu.Unlock()
	if fd := fds.Get(abiType); fd != nil {
		return fd, nil
	}
	fd, err := newFieldDescAndPrefetch(rt)
	if err != nil {
		return nil, err
	}
	fds.Set(abiType, fd)
	return fd, nil
}

var prefetchFieldDescCache = map[reflect.Type]*fieldDesc{}

func newFieldDescAndPrefetch(t reflect.Type) (*fieldDesc, error) {
	if fd := prefetchFieldDescCache[t]; fd != nil {
		return fd, nil
	}
	fd, err := newFieldDesc(t)
	if err != nil {
		return nil, err
	}
	prefetchFieldDescCache[t] = fd
	if err := prefetchSubFieldDesc(fd); err != nil {
		delete(prefetchFieldDescCache, t)
		return nil, err
	}
	return fd, nil
}

func prefetchSubFieldDesc(d *fieldDesc) error {
	for i := range d.fields {
		var t *tType
		f := d.fields[i]
		if f.Type.T == tSTRUCT {
			t = f.Type
		} else if f.Type.T == tMAP && f.Type.V.T == tSTRUCT {
			t = f.Type.V
		} else if f.Type.T == tLIST && f.Type.V.T == tSTRUCT {
			t = f.Type.V
		} else {
			continue
		}
		fd, err := newFieldDescAndPrefetch(t.RT)
		if err != nil {
			return err
		}
		t.Fd = fd
	}
	return nil
}

type iInitDefault interface {
	InitDefault()
}

type fieldDesc struct {
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

	fixedLenFieldSize int       // sum of f.EncodedSize() > 0
	varLenFields      []*tField // list of fields that f.EncodedSize() <= 0
	requiredFields    []*tField
}

func newFieldDesc(t reflect.Type) (*fieldDesc, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errType
	}
	ff, err := defs.ResolveFields(t)
	if err != nil {
		return nil, err
	}
	d := &fieldDesc{rt: t}
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

func (d *fieldDesc) GetField(fid uint16) *tField {
	if fid > d.maxID {
		return nil
	}
	i := d.fieldIdx[fid]
	if i < 0 {
		return nil
	}
	return d.fields[i]
}

func (d *fieldDesc) fromDefsFields(ff []defs.Field) {
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
	d.fields = make([]*tField, len(ff))
	for i, f := range ff {
		d.fields[i] = &tField{}
		d.fields[i].fromDefsField(f)
		d.fieldIdx[f.ID] = i
	}
	d.varLenFields = make([]*tField, 0, len(ff))
	for _, f := range d.fields {
		if n := f.EncodedSize(); n > 0 {
			d.fixedLenFieldSize += n
		} else {
			d.varLenFields = append(d.varLenFields, f)
		}
		if f.Spec == defs.Required {
			d.requiredFields = append(d.requiredFields, f)
		}
	}
}

type tField struct {
	ID     uint16
	Offset uintptr
	Type   *tType

	Opts    defs.Options
	Spec    defs.Requiredness
	Default unsafe.Pointer

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
	f.Opts = x.Opts
	f.Spec = x.Spec

	t := f.Type

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
