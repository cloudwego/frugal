#include "go_asm.h"
#include "funcdata.h"
#include "textflag.h"

TEXT ·callnative1(SB), NOSPLIT, $0-16
    MOVQ fn+0(FP), R12
    MOVQ a0+8(FP), DI
    CALL R12
    RET
