// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT ·SumN(SB),NOSPLIT,$0-16
	MOVQ $0, AX
	MOVQ $0, BX
	MOVQ N+0(FP), CX
loop:
	ADDQ BX, AX
	ADDQ $1, BX
	CMPQ BX, CX
	JLS  loop
	MOVQ AX, ret+8(FP)
	RET