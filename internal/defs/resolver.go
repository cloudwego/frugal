/*
 * Copyright 2021 ByteDance Inc.
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
)

type Requiredness uint

const (
    Default Requiredness = iota
    Required
    Optional
)

type Field struct {
    F    int
    ID   uint16
    Type *Type
    Spec Requiredness
}

func ResolveFields(vt reflect.Type) ([]Field, error) {
    var err error
    var ret []Field

    /* field ID map */
    nfs := vt.NumField()
    ids := make(map[uint64]bool)

    /* traverse all the fields */
    for i := 0; i < nfs; i++ {
        var ok bool
        var id uint64
        var tv string
        var ft []string
        var rx Requiredness
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

        /* check for offser range */
        if sf.Offset > MaxInt56 {
            return nil, fmt.Errorf("field %s.%s offset out of range: %d", vt, sf.Name, sf.Offset)
        }

        /* check for duplicates */
        if !ids[id] {
            ids[id] = true
        } else {
            return nil, fmt.Errorf("duplicated field ID %d for field %s.%s", id, vt, sf.Name)
        }

        /* add to result */
        ret = append(ret, Field {
            F    : int(sf.Offset),
            ID   : uint16(id),
            Spec : rx,
            Type : ParseType(sf.Type, strings.TrimSpace(ft[2])),
        })
    }

    /* sort the field by ID */
    sort.Slice(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
    return ret, nil
}
