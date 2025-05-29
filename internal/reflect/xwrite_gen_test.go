package reflect

import "strings"

func getXWriteCode(typ ttype, t, p string) string {
	t2c := map[ttype]string{
		tBYTE:   "thrift.XBuffer.WriteByte(b, *((*int8)({p})))",
		tI16:    "thrift.XBuffer.WriteI16(b, *((*int16)({p})))",
		tI32:    "thrift.XBuffer.WriteI32(b, *((*int32)({p})))",
		tI64:    "thrift.XBuffer.WriteI64(b, *((*int64)({p})))",
		tDOUBLE: "thrift.XBuffer.WriteI64(b, *((*int64)({p})))",
		tENUM:   "thrift.XBuffer.WriteI32(b, int32(*((*int64)({p}))))",
		tSTRING: "s = *((*string)({p})); thrift.XBuffer.WriteString(b, s)",

		// tSTRUCT, tMAP, tSET, tLIST -> tOTHER
		tOTHER: `if {t}.IsPointer {
		err = {t}.XWriteFunc({t}, b, *(*unsafe.Pointer)({p}))
	} else {
		err = {t}.XWriteFunc({t}, b, {p})
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
