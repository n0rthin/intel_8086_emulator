package main

import (
	"errors"
	"math"
)

// Maps all possible values of the first byte of instruction to the instruction encoding
type InstLookupTable map[byte]InstructionEncoding

func GetInstLookupTable(instTable InstructionTable) InstLookupTable {
	lookupTable := InstLookupTable{}

	for _, enc := range instTable.Encodings {
		firstByteIdx := 0
		bits := 0
		for idx, instBit := range enc.Bits {
			if bits += int(instBit.BitCount); bits > 8 {
				break
			}
			firstByteIdx = idx
		}

		instructionVariants, error := getVariations(enc.Bits[:firstByteIdx])
		if error != nil {
			panic(error)
		}

		for _, variant := range instructionVariants {
			lookupTable[variant] = enc
		}
	}

	return lookupTable
}

func getVariations(instructionBits []InstructionBits) ([]byte, error) {
	return _getVariations(instructionBits, 0, 0)
}

func _getVariations(instructionBits []InstructionBits, variationPrefix byte, bitsSoFar int) ([]byte, error) {
	if len(instructionBits) == 0 {
		return []byte{variationPrefix}, nil
	}

	instBit := instructionBits[0]
	bitsSoFar += int(instBit.BitCount)

	if bitsSoFar > 8 {
		return nil, errors.New("expected total sum of bits (BitCount) in the instructionBits to be 8")
	}

	if instBit.Usage == BitsLiteral {
		nextPrefix := variationPrefix<<instBit.BitCount | instBit.Value
		return _getVariations(instructionBits[1:], nextPrefix, bitsSoFar)
	}

	if instBit.Usage != BitsLiteral && instBit.BitCount > 0 {
		variations := []byte{}
		count := byte(math.Pow(2, float64(instBit.BitCount)))
		for i := byte(0); i < count; i++ {
			nextPrefix := variationPrefix<<instBit.BitCount | i
			_variations, error := _getVariations(instructionBits[1:], nextPrefix, bitsSoFar)
			if error != nil {
				return nil, error
			}

			variations = append(variations, _variations...)
		}

		return variations, nil
	}

	return []byte{}, nil
}
