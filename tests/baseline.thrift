enum Enums {
    ValueA,
    ValueB,
    ValueC,
}

struct Simple {
    1: byte ByteField
    2: i64 I64Field
    3: double DoubleField
    4: i32 I32Field
    5: string StringField
    6: binary BinaryField
    7: Enums enumField
}

struct Nesting {
    1: string String
    2: list<Simple> ListSimple
    3: double Double
    4: i32 I32
    5: list<i32> ListI32
    6: i64 I64
    7: map<string, string> MapStringString
    8: Simple SimpleStruct
    9: map<i32, i64> MapI32I64
    10: list<string> ListString
    11: binary Binary
    12: map<i64, string> MapI64String
    13: list<i64> ListI64
    14: byte Byte
    15: map<string, Simple> MapStringSimple
}

struct Nesting2 {
    1: map<Simple, Nesting> MapSimpleNesting
    2: Simple SimpleStruct
    3: byte Byte
    4: double Double
    5: list<Nesting> ListNesting
    6: i64 I64
    7: Nesting NestingStruct
    8: binary Binary
    9: string String
    11: i32 I32
}

struct DefaultValues {
    1: byte ByteFieldWithDefault = 1
    2: i64 I64FieldWithDefault = 2
    3: double DoubleFieldWithDefault = 3
    4: i32 I32FieldWithDefault = 4
    5: string StringFieldWithDefault = "string field default text"
    6: binary BinaryFieldWithDefault = "binary field default data"
    7: Enums EnumFieldWithDefault = Enums.ValueA
    8: Simple SimpleStructWithDefault = {
        "ByteField": 10,
        "I64Field": 11,
        "DoubleField": 12,
        "I32Field": 13,
        "StringField": "simple string",
        "BinaryField": "simple binary",
        "enumField": Enums.ValueB,
    }
    9: list<i32> ListFieldWithDefault = [1, 2, 3, 4, 5]
    10: set<i32> SetFieldWithDefault = [1, 2, 3, 4, 5]
    11: map<i32, i64> MapI32I64WithDefault = {1: 2, 3: 4}
    12: map<i64, string> MapI64StringWithDefault = {1: "aaa", 2: "bbb"}
    13: map<string, string> MapStringStringWithDefault = {"aaa": "xxx", "bbb": "yyy"}
    14: map<string, Simple> MapStringSimpleWithDefault = {
        "aaa": {
            "ByteField": 20,
            "I64Field": 21,
            "DoubleField": 22,
            "I32Field": 23,
            "StringField": "another simple string",
            "BinaryField": "another simple binary",
            "enumField": Enums.ValueC,
        }
    }
}

struct OptionalDefaultValues {
    1: optional byte ByteFieldWithDefault = 1
    2: optional i64 I64FieldWithDefault = 2
    3: optional double DoubleFieldWithDefault = 3
    4: optional i32 I32FieldWithDefault = 4
    5: optional string StringFieldWithDefault = "string field default text"
    6: optional binary BinaryFieldWithDefault = "binary field default data"
    7: optional Enums EnumFieldWithDefault = Enums.ValueA
    8: optional Simple SimpleStructWithDefault = {
        "ByteField": 10,
        "I64Field": 11,
        "DoubleField": 12,
        "I32Field": 13,
        "StringField": "simple string",
        "BinaryField": "simple binary",
        "enumField": Enums.ValueB,
    }
    9: optional list<i32> ListFieldWithDefault = [1, 2, 3, 4, 5]
    10: optional set<i32> SetFieldWithDefault = [1, 2, 3, 4, 5]
    11: optional map<i32, i64> MapI32I64WithDefault = {1: 2, 3: 4}
    12: optional map<i64, string> MapI64StringWithDefault = {1: "aaa", 2: "bbb"}
    13: optional map<string, string> MapStringStringWithDefault = {"aaa": "xxx", "bbb": "yyy"}
    14: optional map<string, Simple> MapStringSimpleWithDefault = {
        "aaa": {
            "ByteField": 20,
            "I64Field": 21,
            "DoubleField": 22,
            "I32Field": 23,
            "StringField": "another simple string",
            "BinaryField": "another simple binary",
            "enumField": Enums.ValueC,
        }
    }
}
