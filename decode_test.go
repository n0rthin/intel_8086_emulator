package main

import (
	"errors"
	"reflect"
	"testing"
)

type TestCaseDecode struct {
	name        string
	bytes       []byte
	instruction Instruction
}

var testCasesDecode = []TestCaseDecode{
	{
		"register to register, DW=10",
		[]byte{0b10001010, 0b11000100},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 1}},
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 1, 1}},
			},
		},
	},
	{
		"register to register, DW=00",
		[]byte{0b10001000, 0b11000100},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 1, 1}},
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 1}},
			},
		},
	},
	{
		"register to register, DW=11",
		[]byte{0b10001011, 0b11000100},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 2}},
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegSP, 0, 2}},
			},
		},
	},
	{
		"register to register, DW=01",
		[]byte{0b10001001, 0b11000100},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegSP, 0, 2}},
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 2}},
			},
		},
	},
	{
		"memory to register, DW=10, MOD=00, 16-bit displacement, direct address",
		[]byte{0b10001010, 0b00000110, 0b11110000, 0b10101010},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 1}},
				{Type: OperandMemory, MemAccess: MemAccess{[]RegisterAccess{}, 0b1010101011110000}},
			},
		},
	},
	{
		"memory to register, DW=10, MOD=00, no displacement",
		[]byte{0b10001010, 0b00000000},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 1}},
				{Type: OperandMemory, MemAccess: MemAccess{[]RegisterAccess{RegAccessBX, RegAccessSI}, 0}},
			},
		},
	},
	{
		"memory to register, DW=10, MOD=01, 8-bit displacement",
		[]byte{0b10001010, 0b01000001, 0b11010001},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 1}},
				{Type: OperandMemory, MemAccess: MemAccess{[]RegisterAccess{RegAccessBX, RegAccessDI}, 0b11010001}},
			},
		},
	},
	{
		"memory to register, DW=10, MOD=10, 16-bit displacement",
		[]byte{0b10001010, 0b10000100, 0b11100001, 0b00011000},
		Instruction{
			OpMov,
			[2]InstOperand{
				{Type: OperandRegister, RegisterAccess: RegisterAccess{RegA, 0, 1}},
				{Type: OperandMemory, MemAccess: MemAccess{[]RegisterAccess{RegAccessSI}, 0b0001100011100001}},
			},
		},
	},
}

type BitReaderTest struct {
	Bytes   []byte
	currBit uint
}

func (br *BitReaderTest) ReadNBits(n byte) (byte, error) {
	currBitInsideByte := br.currBit % 8
	if currBitInsideByte+uint(n) > 8 {
		return 0, errors.New("trying to read over bytes boundary")
	}

	currByte := br.Bytes[br.currBit/8]
	br.currBit += uint(n)
	return currByte << byte(currBitInsideByte) >> (8 - n), nil
}

func (br *BitReaderTest) ReadNBytes(n uint) ([]byte, error) {
	currBitInsideByte := br.currBit % 8
	if currBitInsideByte != 0 {
		return []byte{}, errors.New("cannot perform byte-level read")
	}

	currByte := br.currBit / 8
	br.currBit += n * 8
	return br.Bytes[currByte : currByte+n], nil
}

func TestDecodeTable(t *testing.T) {
	lookupTable := GetInstLookupTable(&InstructionTable8086)

	for _, tc := range testCasesDecode {
		t.Run(tc.name, func(t *testing.T) {
			bitReader := BitReaderTest{tc.bytes, 0}

			enc, ok := lookupTable[tc.bytes[0]]
			if !ok {
				t.Errorf("lookup table doesn't have 0b%08b variant", tc.bytes[0])
				return
			}

			instruction, err := TryDecode(&bitReader, enc)
			if err != nil {
				t.Errorf(`got error %v`, err)
			}

			if !reflect.DeepEqual(instruction, tc.instruction) {
				t.Errorf("result != expected.\nresult:   %+v\nexpected: %+v", instruction, tc.instruction)
			}
		})
	}
}
