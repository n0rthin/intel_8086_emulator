package main

import (
	"fmt"
	"sim86/utils"
)

/*
  This instruction table is a direct translation of table 4-12 in the Intel 8086 manual.
*/

// Types

type InstructionTable struct {
	Encodings     []InstructionEncoding
	EncodingCount uint
}

type InstructionEncoding struct {
	Op   OperationType
	Bits [16]InstructionBits
}

type OperationType byte

const (
	OpNone OperationType = iota
	OpMov
	OpAdd
	OpSub
)

type InstructionBits struct {
	Usage    InstructionBitsUsage
	BitCount byte
	Value    byte
}

type InstructionBitsUsage byte

const (
	BitsEnd InstructionBitsUsage = iota
	BitsLiteral
	BitsD
	BitsS
	BitsW
	BitsV
	BitsZ
	BitsMOD
	BitsREG
	BitsRM
	BitsSR
	BitsDisp
	BitsData
	BitsDispAlwaysW  // Tag for instructions where the displacement is always 16 bits
	BitsWMakesDataW  // Tag for instructions where SW=01 makes the data field become 16 bits
	BitsRMRegAlwaysW // Tag for instructions where the register encoded in RM is always 16-bit width
	BitsRelJMPDisp   // Tag for instructions that require address adjustment to go through NASM properly
	BitsFar          // Tag for instructions that require a "far" keyword in their ASM to select the right opcode
	BitsCount
)

// Table

func INST(op OperationType, bits ...InstructionBits) InstructionEncoding {
	utils.Assert(
		len(bits) > 16,
		fmt.Sprintf("Expected 16 bits at most, received %d instead. Op %d", len(bits), op),
	)

	var bits16 [16]InstructionBits
	copy(bits16[:], bits)

	return InstructionEncoding{Op: op, Bits: bits16}
}

func B(value byte, bitCount byte) InstructionBits {
	return InstructionBits{Usage: BitsLiteral, BitCount: bitCount, Value: value}
}

// Imp = implicit

func ImpW(value byte) InstructionBits {
	return InstructionBits{Usage: BitsW, BitCount: 0, Value: value}
}

func ImpD(value byte) InstructionBits {
	return InstructionBits{Usage: BitsD, BitCount: 0, Value: value}
}

func ImpREG(value byte) InstructionBits {
	return InstructionBits{Usage: BitsREG, BitCount: 0, Value: value}
}

func ImpMOD(value byte) InstructionBits {
	return InstructionBits{Usage: BitsMOD, BitCount: 0, Value: value}
}

func ImpRM(value byte) InstructionBits {
	return InstructionBits{Usage: BitsRM, BitCount: 0, Value: value}
}

var D = InstructionBits{Usage: BitsD, BitCount: 1}
var S = InstructionBits{Usage: BitsS, BitCount: 1}
var W = InstructionBits{Usage: BitsW, BitCount: 1}
var RM = InstructionBits{Usage: BitsRM, BitCount: 3}
var MOD = InstructionBits{Usage: BitsMOD, BitCount: 2}
var REG = InstructionBits{Usage: BitsREG, BitCount: 3}
var SR = InstructionBits{Usage: BitsSR, BitCount: 2}
var ADDR = InstructionBits{Usage: BitsDisp, BitCount: 0, Value: 0}
var DISP_ALWAYS_W = InstructionBits{Usage: BitsDispAlwaysW, BitCount: 0, Value: 1}
var DATA = InstructionBits{Usage: BitsData, BitCount: 0}
var DATA_IF_W = InstructionBits{Usage: BitsWMakesDataW, BitCount: 0, Value: 1}

var encodings []InstructionEncoding = []InstructionEncoding{
	INST(OpMov, B(0b00100010, 6), D, W, MOD, REG, RM),                                                  // register/memory to/from register
	INST(OpMov, B(0b01100011, 7), W, MOD, B(0b000, 3), RM, DATA, DATA_IF_W, ImpD(0)),                   // immediate to register/memory
	INST(OpMov, B(0b00001011, 4), W, REG, DATA, DATA_IF_W, ImpD(1)),                                    // immediate to register
	INST(OpMov, B(0b01010000, 7), W, ADDR, DISP_ALWAYS_W, ImpREG(0), ImpMOD(0), ImpRM(0b110), ImpD(1)), // memory to accumulator
	INST(OpMov, B(0b01010001, 7), W, ADDR, ImpREG(0), ImpMOD(0), ImpRM(0b110), ImpD(0)),                // accumulator to memory
	INST(OpMov, B(0b00100011, 6), D, B(0, 1), MOD, B(0, 1), SR, RM, ImpW(1)),                           // register/memory to segment register and vice versa; This collapses 2 entries in the 8086 table by adding an explicit D bit
	INST(OpAdd, B(0b00000000, 6), D, W, MOD, REG, RM),                                                  // reg/memory with register to either
	INST(OpAdd, B(0b00100000, 6), S, W, MOD, B(0b000, 3), RM, DATA, DATA_IF_W),                         // immediate to register/memory
	INST(OpAdd, B(0b00000010, 7), W, DATA, DATA_IF_W, ImpREG(0), ImpD(1)),                              // immediate to accumulator
	INST(OpSub, B(0b00001010, 6), D, W, MOD, REG, RM),                                                  // reg/memory with register to either
	INST(OpSub, B(0b00100000, 6), S, W, MOD, B(0b101, 3), RM, DATA, DATA_IF_W),                         // immediate to register/memory
	INST(OpSub, B(0b00010110, 7), W, DATA, DATA_IF_W, ImpREG(0), ImpD(1)),                              // immediate to accumulator
}

var InstructionTable8086 = InstructionTable{
	Encodings:     encodings,
	EncodingCount: uint(len(encodings)),
}
