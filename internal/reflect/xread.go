package reflect

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/bytedance/gopkg/lang/dirtmake"
	"github.com/cloudwego/gopkg/protocol/thrift"
	"github.com/cloudwego/gopkg/xbuf"

	"github.com/cloudwego/frugal/internal/defs"
)

func (d *tDecoder) XRead(b *xbuf.XReadBuffer, base unsafe.Pointer, sd *structDesc, maxdepth int) (err error) {
	if maxdepth == 0 {
		return errDepthLimitExceeded
	}
	var bs *bitset
	if len(sd.requiredFieldIDs) > 0 {
		bs = bitsetPool.Get().(*bitset)
		defer bitsetPool.Put(bs)
		for _, f := range sd.requiredFieldIDs {
			bs.unset(f)
		}
	}

	var ufs []byte
	for {
		tp := ttype(b.ReadN(1)[0])
		if tp == tSTOP {
			break
		}
		fid := binary.BigEndian.Uint16(b.ReadN(2))

		f := sd.GetField(fid)
		if f == nil || f.Type.WT != tp {
			ufs, err = thrift.XBuffer.Skip(b, thrift.TType(tp), ufs, sd.hasUnknownFields)
			if err != nil {
				return fmt.Errorf("skip unknown field %d of struct %s err: %w", fid, sd.rt.String(), err)
			}
			continue
		}
		p := unsafe.Add(base, f.Offset) // pointer to the field

		t := f.Type
		p = d.mallocIfPointer(t, p)
		if t.FixedSize > 0 {
			xreadFixedSizeTypes(t.T, b, p)
		} else {
			err = d.xreadType(t, b, p, maxdepth-1)
			if err != nil {
				return fmt.Errorf("decode field %d of struct %s err: %w", fid, sd.rt.String(), err)
			}
		}
		if bs != nil {
			bs.set(f.ID)
		}
	}
	for _, fid := range sd.requiredFieldIDs {
		if !bs.test(fid) {
			return newRequiredFieldNotSetException(lookupFieldName(sd.rt, sd.GetField(fid).Offset))
		}
	}
	if len(ufs) > 0 {
		*(*[]byte)(unsafe.Add(base, sd.unknownFieldsOffset)) = ufs
	}
	return nil
}

func xreadFixedSizeTypes(t ttype, b *xbuf.XReadBuffer, p unsafe.Pointer) {
	switch t {
	case tBOOL, tBYTE:
		*(*byte)(p) = b.ReadN(1)[0] // XXX: for tBOOL 1->true, 2->true/false
	case tDOUBLE, tI64:
		*(*uint64)(p) = binary.BigEndian.Uint64(b.ReadN(8))
	case tI16:
		*(*int16)(p) = int16(binary.BigEndian.Uint16(b.ReadN(2)))
	case tI32:
		*(*int32)(p) = int32(binary.BigEndian.Uint32(b.ReadN(4)))
	case tENUM:
		*(*int64)(p) = int64(int32(binary.BigEndian.Uint32(b.ReadN(4))))
	default:
		panic("bug")
	}
}

func (d *tDecoder) xreadType(t *tType, b *xbuf.XReadBuffer, p unsafe.Pointer, maxdepth int) (err error) {
	if maxdepth == 0 {
		err = errDepthLimitExceeded
		return
	}
	if t.FixedSize > 0 {
		xreadFixedSizeTypes(t.T, b, p)
		return
	}
	switch t.T {
	case tSTRING:
		l := int(binary.BigEndian.Uint32(b.ReadN(4)))
		if l < 0 {
			err = errNegativeSize
			return
		}
		if l == 0 {
			if t.Tag == defs.T_binary {
				*(*[]byte)(p) = []byte{}
			} else {
				*(*string)(p) = ""
			}
			return
		}
		buf := dirtmake.Bytes(l, l)
		b.CopyBytes(buf)

		if t.Tag == defs.T_binary {
			h := (*sliceHeader)(p)
			h.Data = uintptr(unsafe.Pointer(&buf[0]))
			h.Len = l
			h.Cap = l
		} else { //  convert to str
			h := (*stringHeader)(p)
			h.Data = uintptr(unsafe.Pointer(&buf[0]))
			h.Len = l
		}
		return
	case tMAP:
		// map header
		buf := b.ReadN(6)
		t0, t1, l := ttype(buf[0]), ttype(buf[1]), int(binary.BigEndian.Uint32(buf[2:]))
		if l < 0 {
			return errNegativeSize
		}

		// check types
		kt := t.K
		vt := t.V
		if t0 != kt.WT || t1 != vt.WT {
			return newTypeMismatchKV(kt.WT, vt.WT, t0, t1)
		}

		// decode map

		// tmp vars
		// tmpk = decode(b)
		// tmpv = decode(b)
		// map[tmpk] = tmpv
		tmp := t.MapTmpVarsPool.Get().(*tmpMapVars)
		k := tmp.k
		v := tmp.v
		kp := tmp.kp
		vp := tmp.vp
		m := reflect.MakeMapWithSize(t.RT, l)
		*((*uintptr)(p)) = m.Pointer() // p = make(t.RT, l)

		// pre-allocate space for keys and values if they're pointers
		// like
		// kp = &sliceK[i], decode(b, kp)
		// instead of
		// kp = new(type), decode(b, kp)
		var sliceK unsafe.Pointer
		if kt.IsPointer && l > 0 {
			sliceK = d.Malloc(l*kt.V.Size, kt.V.Align, kt.V.MallocAbiType)
		}
		var sliceV unsafe.Pointer
		if vt.IsPointer && l > 0 {
			sliceV = d.Malloc(l*vt.V.Size, vt.V.Align, vt.V.MallocAbiType)
		}

		for j := 0; j < l; j++ {
			p = kp
			if kt.IsPointer { // p = &sliceK[j]
				if j != 0 {
					sliceK = unsafe.Add(sliceK, kt.V.Size) // next
				}
				*(*unsafe.Pointer)(p) = sliceK
				p = sliceK
			}
			if kt.FixedSize > 0 {
				xreadFixedSizeTypes(kt.T, b, p)
			} else {
				if err = d.xreadType(kt, b, p, maxdepth-1); err != nil {
					break
				}
			}
			p = vp
			if vt.IsPointer { // p = &sliceV[j]
				if j != 0 { // next
					sliceV = unsafe.Add(sliceV, vt.V.Size)
				}
				*(*unsafe.Pointer)(p) = sliceV
				p = sliceV
			}
			if vt.FixedSize > 0 {
				xreadFixedSizeTypes(vt.T, b, p)
			} else {
				if err = d.xreadType(vt, b, p, maxdepth-1); err != nil {
					break
				}
			}
			m.SetMapIndex(k, v)
		}
		t.MapTmpVarsPool.Put(tmp) // no defer, it may be in hot path
		return nil
	case tLIST, tSET: // NOTE: for tSET, it may be map in the future
		// list header
		buf := b.ReadN(5)
		tp, l := ttype(buf[0]), int(binary.BigEndian.Uint32(buf[1:]))
		if l < 0 {
			return errNegativeSize
		}
		// check types
		et := t.V
		if et.WT != tp {
			return newTypeMismatch(et.WT, tp)
		}
		// decode list
		h := (*sliceHeader)(p) // update the slice field
		if l <= 0 {
			h.Zero()
			return nil
		}
		x := d.Malloc(l*et.Size, et.Align, et.MallocAbiType) // malloc for slice. make([]Type, l, l)
		h.Data = uintptr(x)
		h.Len = l
		h.Cap = l

		// pre-allocate space for elements if they're pointers
		// like
		// v[i] = &sliceData[i]
		// instead of
		// v[i] = new(type)
		var sliceData unsafe.Pointer
		if et.IsPointer {
			sliceData = d.Malloc(l*et.V.Size, et.V.Align, et.V.MallocAbiType)
		}

		p = x // point to the 1st element, and then decode one by one
		for j := 0; j < l; j++ {
			if j != 0 {
				p = unsafe.Add(p, et.Size) // next element
			}
			vp := p // v[j]

			// p = &sliceData[j], see comment of sliceData above
			if et.IsPointer {
				if j != 0 {
					sliceData = unsafe.Add(sliceData, et.V.Size) // next
				}
				*(*unsafe.Pointer)(p) = sliceData // v[j] = &sliceData[i]
				vp = sliceData                    // &v[j]
			}

			if et.FixedSize > 0 {
				xreadFixedSizeTypes(et.T, b, vp)
			} else {
				err = d.xreadType(et, b, vp, maxdepth-1)
				if err != nil {
					return err
				}
			}
		}
		return nil
	case tSTRUCT:
		if t.Sd.hasInitFunc {
			f := t.Sd.initFunc // copy on write, reuse itab of iface
			updateIface(unsafe.Pointer(&f), p)
			f.InitDefault()
		}
		return d.XRead(b, p, t.Sd, maxdepth-1)
	}
	return fmt.Errorf("unknown type: %d", t.T)
}
