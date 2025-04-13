package main

import (
	"testing"
)

func TestBitsLimit(t *testing.T) {
	instructionBits := []InstructionBits{{Usage: BitsLiteral, BitCount: 9}}
	_, err := getVariations(instructionBits)
	if err == nil {
		t.Errorf(`Expected getVariations to return error if total bits count is > 8`)
	}
}

type TestCaseLookupTable struct {
	name            string
	instructionBits []InstructionBits
	variations      []byte
}

var testCases = []TestCaseLookupTable{
	{
		"register/memory to/from register DW=10",
		BITS(B(0b00100010, 6), D, W),
		[]byte{0b10001000, 0b10001001, 0b10001010, 0b10001011},
	},
	{"one 8 bit literal", BITS(B(0b11111111, 8)), []byte{0b11111111}},
	{"two 4 bit literals", BITS(B(0b1111, 4), B(0b1001, 4)), []byte{0b11111001}},
	{
		"one 6 bit literal and two 1 bit non-literals",
		BITS(B(0b111111, 6), D, W),
		[]byte{0b11111100, 0b11111101, 0b11111110, 0b11111111},
	},
	{
		"one 6 bit literal, one 1 bit non-literal, one 1 bit literal",
		BITS(B(0b111111, 6), D, B(0b1, 1)),
		[]byte{0b11111101, 0b11111111},
	},
}

func BITS(bits ...InstructionBits) []InstructionBits {
	return bits
}

func TestLookupTable(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			variations, err := getVariations(tc.instructionBits)
			if err != nil {
				t.Errorf(`Got error %v`, err)
			}

			if len(variations) != len(tc.variations) {
				t.Errorf(`got %d variations, expected %d`, len(variations), len(tc.variations))
			}

			for idx, variation := range variations {
				if variation != tc.variations[idx] {
					t.Errorf(`#%d: 0b%08b (result) != 0b%08b (expected)`, idx, variation, tc.variations[idx])
				}
			}
		})
	}
}

type TestCasesGetFirstNBitsIdx struct {
	name   string
	bits   []InstructionBits
	n      uint
	result uint
}

var testCasesGetFirstNBitsIdx = []TestCasesGetFirstNBitsIdx{
	{
		name:   "",
		bits:   BITS(B(0b00100010, 6), D, W, MOD, REG, RM),
		n:      8,
		result: 3,
	},
}

func TestGetFirstNBitsIdx(t *testing.T) {
	for _, tc := range testCasesGetFirstNBitsIdx {
		t.Run(tc.name, func(t *testing.T) {
			idx := getFirstNBitsIdx(tc.bits, tc.n)
			if idx != tc.result {
				t.Errorf(`result != expected, %d != %d`, idx, tc.result)
			}
		})
	}
}
