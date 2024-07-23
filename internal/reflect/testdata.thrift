namespace go reflect

enum Numberz
{
  TEN = 10
}

typedef i64 UserID

struct Msg
{
  1: string message;
  2: i32 type;
}

struct TestTypes {
  1: required bool FBool;
  2: byte FByte;
  3: i8 I8;
  4: i16 I16;
  5: i32 I32;
  6: i64 I64;
  7: double Double;
  8: string String;
  9: binary Binary;
  10: Numberz Enum;
  11: UserID UID;
  12: Msg S;
  20: required map<i32, i32> M0;
  21: map<i32, string> M1;
  22: map<i32, Msg> M2;
  23: map<string, Msg> M3;
  30: required list<i32> L0;
  31: list<string> L1;
  32: list<Msg> L2;
  40: required set<i32> S0;
  41: set<string> S1;
  50: list<map<i32, i32>> LM;
  60: map<i32, list<i32>> ML;
}

struct TestTypesOptional {
  1: optional bool FBool;
  2: optional byte FByte;
  3: optional i8 I8;
  4: optional i16 I16;
  5: optional i32 I32;
  6: optional i64 I64;
  7: optional double Double;
  8: optional string String;
  9: optional binary Binary;
  10: optional Numberz Enum;
  11: optional UserID UID;
  12: optional Msg S;
  20: optional map<i32, i32> M0;
  21: optional map<i32, string> M1;
  22: optional map<i32, Msg> M2;
  23: optional map<string, Msg> M3;
  30: optional list<i32> L0;
  31: optional list<string> L1;
  32: optional list<Msg> L2;
  40: optional set<i32> S0;
  41: optional set<string> S1;
  50: optional list<map<i32, i32>> LM;
  60: optional map<i32, list<i32>> ML;
}

struct TestTypesWithDefault {
  1: optional bool FBool = true;
  2: optional byte FByte = 2;
  3: optional i8 I8 = 3;
  4: optional i16 I6 = 4;
  5: optional i32 I32 = 5;
  6: optional i64 I64 = 6;
  7: optional double Double = 7;
  8: optional string String = "8";
  9: optional binary Binary = "8";
  10: optional Numberz Enum = 10;
  11: optional UserID UID = 11;
  30: optional list<i32> L0 = [ 30 ];
  40: optional set<i32> S0 = [ 40 ];
}

struct TestTypesForBenchmark {
  1: optional bool B0 = true;
  2: optional bool B1;
  3: required bool B2;
  11: optional string Str0 = "8";
  12: optional string Str1;
  13: required string Str2;
  14: required string Str3 = "9";
  21: optional Msg Msg0;
  22: required Msg Msg1;
  31: optional map<i32, i32> M0;
  32: required map<string, Msg> M1;
  41: optional list<i32> L0;
  42: required list<Msg> L1;
  51: optional set<i32> Set0;
  52: required set<string> Set1;
}

