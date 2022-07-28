/*
 * Copyright 2022 ByteDance Inc.
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

package defs

import (
    `fmt`
    `reflect`
    `sort`
    `strconv`
    `strings`
    `sync`
)

type (
	Requiredness uint8
)

const (
    Default Requiredness = iota
    Required
    Optional
)

type Field struct {
    F       int
    ID      uint16
    Type    *Type
    Spec    Requiredness
    Default reflect.Value
}

var (
    fieldsLock  = new(sync.RWMutex)
    fieldsCache = make(map[reflect.Type][]Field)
)

func ResolveFields(vt reflect.Type) ([]Field, error) {
    var ok bool
    var ex error
    var fv []Field

    /* attempt to find in cache */
    fieldsLock.RLock()
    fv, ok = fieldsCache[vt]
    fieldsLock.RUnlock()

    /* check if it exists */
    if ok {
        return fv, nil
    }

    /* retry with write lock */
    fieldsLock.Lock()
    defer fieldsLock.Unlock()

    /* try again */
    if fv, ok = fieldsCache[vt]; ok {
        return fv, nil
    }

    /* still not found, do the actual resolving */
    if fv, ex = doResolveFields(vt); ex != nil {
        return nil, ex
    }

    /* update cache */
    fieldsCache[vt] = fv
    return fv, nil
}

func doResolveFields(vt reflect.Type) ([]Field, error) {
    var err error
    var ret []Field
    var mem reflect.Value

    /* field ID map and default values */
    val := reflect.New(vt)
    ids := make(map[uint64]struct{}, vt.NumField())

    /* check for default values */
    if def, ok := val.Interface().(DefaultInitializer); ok {
        mem = val.Elem()
        def.InitDefault()
    }

    /* traverse all the fields */
    for i := 0; i < vt.NumField(); i++ {
        var ok bool
        var pt *Type
        var id uint64
        var tv string
        var ft []string
        var rx Requiredness
        var rv reflect.Value
        var sf reflect.StructField

        /* extract the field, ignore anonymous or private fields */
        if sf = vt.Field(i); sf.Anonymous || sf.PkgPath != "" {
            continue
        }

        /* ignore fields that does not declare the "frugal" tag */
        if tv, ok = sf.Tag.Lookup("frugal"); !ok {
            continue
        }

        /* must have 3 fields: ID, Requiredness, Type */
        if ft = strings.Split(tv, ","); len(ft) != 3 {
            return nil, fmt.Errorf("invalid tag for field %s.%s", vt, sf.Name)
        }

        /* parse the field index */
        if id, err = strconv.ParseUint(strings.TrimSpace(ft[0]), 10, 16); err != nil {
            return nil, fmt.Errorf("invalid field number for field %s.%s: %w", vt, sf.Name, err)
        }

        /* convert the requiredness of this field */
        switch strings.TrimSpace(ft[1]) {
            case "default"  : rx = Default
            case "required" : rx = Required
            case "optional" : rx = Optional
            default         : return nil, fmt.Errorf("invalid requiredness for field %s.%s", vt, sf.Name)
        }

        /* check for duplicates */
        if _, ok = ids[id]; !ok {
            ids[id] = struct{}{}
        } else {
            return nil, fmt.Errorf("duplicated field ID %d for field %s.%s", id, vt, sf.Name)
        }

        /* only optional fields or structs can be pointers */
        if pt = ParseType(sf.Type, strings.TrimSpace(ft[2])); rx != Optional && pt.T == T_pointer && pt.V.T != T_struct {
            return nil, fmt.Errorf("only optional fields or structs can be pointers, not %s: %s.%s", sf.Type, vt, sf.Name)
        }

        /* check for nested pointers */
        if pt.T == T_pointer && pt.V.T == T_pointer {
            return nil, fmt.Errorf("struct fields cannot have nested pointers: %s.%s", vt, sf.Name)
        }

        /* get the default value if any */
        if mem.IsValid() {
            rv = mem.FieldByIndex(sf.Index)
        }

        /* add to result */
        ret = append(ret, Field {
            F       : int(sf.Offset),
            ID      : uint16(id),
            Type    : pt,
            Spec    : rx,
            Default : rv,
        })
    }

    /* sort the field by ID */
    sort.Slice(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
    return ret, nil
}
