package main

import (
	"errors"
	"fmt"
)

type Instruction struct {
	Op       OperationType
	Operands [2]InstOperand // the first operand is always destination
}

type InstOperand struct {
	Type           OperandType
	RegisterAccess RegisterAccess
	MemAccess      MemAccess
	// immediate
}

type RegisterAccess struct {
	Reg    Register
	Offset uint
	Count  uint
}

type MemAccess struct {
	Regs []RegisterAccess
	Disp uint16
}

type OperandType byte

const (
	OperandRegister OperandType = iota
	OperandMemory
	OperandImmediate
)

type Register byte

const (
	RegA Register = iota
	RegB
	RegC
	RegD
	RegSP
	RegBP
	RegSI
	RegDI
	RegES
	RegCS
	RegSS
	RegDS
)

var (
	RegAccessAL = RegisterAccess{RegA, 0, 1}
	RegAccessAH = RegisterAccess{RegA, 1, 1}
	RegAccessAX = RegisterAccess{RegA, 0, 2}
	RegAccessCL = RegisterAccess{RegC, 0, 1}
	RegAccessCH = RegisterAccess{RegC, 1, 1}
	RegAccessCX = RegisterAccess{RegC, 0, 2}
	RegAccessDL = RegisterAccess{RegD, 0, 1}
	RegAccessDH = RegisterAccess{RegD, 1, 1}
	RegAccessDX = RegisterAccess{RegD, 0, 2}
	RegAccessBL = RegisterAccess{RegB, 0, 1}
	RegAccessBH = RegisterAccess{RegB, 1, 1}
	RegAccessBX = RegisterAccess{RegB, 0, 2}
	RegAccessSP = RegisterAccess{RegSP, 0, 2}
	RegAccessBP = RegisterAccess{RegBP, 0, 2}
	RegAccessSI = RegisterAccess{RegSI, 0, 2}
	RegAccessDI = RegisterAccess{RegDI, 0, 2}
)

// translation of the table 4-10 (MOD=11) in the Intel 8086 manual
var RegTable = [8][2]RegisterAccess{
	{RegAccessAL, RegAccessAX},
	{RegAccessCL, RegAccessCX},
	{RegAccessDL, RegAccessDX},
	{RegAccessBL, RegAccessBX},
	{RegAccessAH, RegAccessSP},
	{RegAccessCH, RegAccessBP},
	{RegAccessDH, RegAccessSI},
	{RegAccessBH, RegAccessDI},
}

// translation of the table 4-10 (MOD=00, 01, 10) in the Intel 8086 manual
var EffectiveAddrTable = [8]MemAccess{
	{[]RegisterAccess{RegAccessBX, RegAccessSI}, 0},
	{[]RegisterAccess{RegAccessBX, RegAccessDI}, 0},
	{[]RegisterAccess{RegAccessBP, RegAccessSI}, 0},
	{[]RegisterAccess{RegAccessBP, RegAccessDI}, 0},
	{[]RegisterAccess{RegAccessSI}, 0},
	{[]RegisterAccess{RegAccessDI}, 0},
	{[]RegisterAccess{RegAccessBP}, 0},
	{[]RegisterAccess{RegAccessDX}, 0},
}

// translation of the table 4-8 in the Intel 8086 manual
type Mode byte

const (
	MOD_MemNoDisp Mode = iota // 0b00
	MOD_Mem8Bit               // 0b01
	MOD_Mem16Bit              // 0b10
	MOD_Reg                   // 0b11
)

var DispSizeBytes = map[Mode]uint{
	MOD_MemNoDisp: 0, MOD_Mem8Bit: 1,
	MOD_Mem16Bit: 2, MOD_Reg: 0,
}

func isDirectAddress(mod Mode, rm byte) bool {
	return mod == MOD_MemNoDisp && rm == 0b110
}

func getMemAccess(mod Mode, rm byte) MemAccess {
	memAccess := EffectiveAddrTable[rm]
	if isDirectAddress(mod, rm) {
		memAccess.Regs = []RegisterAccess{}
	}
	return memAccess
}

func getDispSizeBytes(mod Mode, rm byte) uint {
	if isDirectAddress(mod, rm) {
		return 2
	}

	return DispSizeBytes[mod]
}

type BitReader interface {
	// Peek(n byte) (byte, error)
	// ReadBit() (byte, error)
	ReadNBits(n byte) (byte, error)
	ReadNBytes(n uint) ([]byte, error)
}

func TryDecode(reader BitReader, instEncoding InstructionEncoding) (Instruction, error) {
	instruction := Instruction{Op: instEncoding.Op}
	var D, W, RM byte
	var MOD Mode
	var registerAccess InstOperand
	REG := -1

	for _, encBit := range instEncoding.Bits {
		var bits byte
		if encBit.BitCount == 0 {
			bits = encBit.Value
		} else {
			bitsFromReader, err := reader.ReadNBits(encBit.BitCount)
			if err != nil {
				return Instruction{}, err
			}
			bits = bitsFromReader
		}

		switch encBit.Usage {
		case BitsD:
			D = bits
			if D < 0 || D > 1 {
				return Instruction{}, errors.New(fmt.Sprintf("Expected D to be 0 <= D <= 1, got %d", D))
			}
		case BitsW:
			W = bits
			if W < 0 || W > 1 {
				return Instruction{}, errors.New(fmt.Sprintf("Expected W to be 0 <= W <= 1, got %d", W))
			}
		case BitsMOD:
			MOD = Mode(bits)
			if MOD < 0 || MOD > 3 {
				return Instruction{}, errors.New(fmt.Sprintf("Expected MOD to be 0 <= MOD <= 3, got %d", MOD))
			}
		case BitsREG:
			REG = int(bits)
			if REG < 0 || int(REG) >= len(RegTable) {
				return Instruction{}, errors.New(fmt.Sprintf("Expected REG to be 0 <= REG <= %d, got %d", len(RegTable)-1, REG))
			}
		case BitsRM:
			RM = bits
			if RM < 0 || int(RM) >= len(RegTable) {
				return Instruction{}, errors.New(fmt.Sprintf("Expected RM to be 0 <= RM <= %d, got %d", len(RegTable)-1, RM))
			}
		}
	}

	if REG >= 0 {
		registerAccess = InstOperand{Type: OperandRegister, RegisterAccess: RegTable[REG][W]}
	}

	switch MOD {
	case MOD_MemNoDisp:
		fallthrough
	case MOD_Mem8Bit:
		fallthrough
	case MOD_Mem16Bit:
		memAccess := getMemAccess(MOD, RM)
		dispSize := getDispSizeBytes(MOD, RM)
		dispBytes, err := reader.ReadNBytes(dispSize)
		if err != nil {
			return Instruction{}, err
		}

		// displacement is encoded in the little-endian order
		for idx, b := range dispBytes {
			memAccess.Disp = memAccess.Disp | uint16(b)<<(idx*8)
		}

		memAccessOperand := InstOperand{Type: OperandMemory, MemAccess: memAccess}
		if D == 1 {
			instruction.Operands[0] = registerAccess
			instruction.Operands[1] = memAccessOperand
		} else {
			instruction.Operands[0] = memAccessOperand
			instruction.Operands[1] = registerAccess
		}
	case MOD_Reg:
		rmRegisterAccess := InstOperand{Type: OperandRegister, RegisterAccess: RegTable[RM][W]}
		if D == 1 {
			instruction.Operands[0] = registerAccess
			instruction.Operands[1] = rmRegisterAccess
		} else {
			instruction.Operands[0] = rmRegisterAccess
			instruction.Operands[1] = registerAccess
		}
	}

	return instruction, nil
}
