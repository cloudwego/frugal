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
    10: set<Nesting> SetNesting
    11: i32 I32
}
