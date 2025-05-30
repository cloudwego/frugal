package reflect

import "strings"

func getGridWriteCode(typ ttype, t, p string) string {
	t2c := map[ttype]string{
		tBYTE:   "b.MallocN(1)[0] = *(*byte)({p})",
		tI16:    "binary.BigEndian.PutUint16(b.MallocN(2), *((*uint16)({p})))",
		tI32:    "binary.BigEndian.PutUint32(b.MallocN(4), *((*uint32)({p})))",
		tI64:    "binary.BigEndian.PutUint64(b.MallocN(8), *((*uint64)({p})))",
		tDOUBLE: "binary.BigEndian.PutUint64(b.MallocN(8), *((*uint64)({p})))",
		tENUM:   "binary.BigEndian.PutUint32(b.MallocN(4), uint32(*((*int64)({p}))))",
		tSTRING: `s = *((*string)({p}))
				if len(s) < nocopyWriteThreshold {
					buf := b.MallocN(len(s) + 4)
					binary.BigEndian.PutUint32(buf, uint32(len(s)))
					copy(buf[4:], s)
				} else {
					binary.BigEndian.PutUint32(b.MallocN(4), uint32(len(s)))
					b.WriteDirect(unsafex.StringToBinary(s))
				}`,

		// tSTRUCT, tMAP, tSET, tLIST -> tOTHER
		tOTHER: `if {t}.IsPointer {
		err = {t}.GridWriteFunc({t}, b, *(*unsafe.Pointer)({p}))
	} else {
		err = {t}.GridWriteFunc({t}, b, {p})
	}
	if err != nil {
		return err
}`,
	}
	s, ok := t2c[typ]
	if !ok {
		panic("type doesn't have code: " + ttype2str(typ))
	}
	s = strings.ReplaceAll(s, "{t}", t)
	s = strings.ReplaceAll(s, "{p}", p)
	return s
}
