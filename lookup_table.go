package main

import (
	"errors"
	"math"
)

// Maps all possible values of the first byte of instruction to the instruction encoding
type InstLookupTable map[uint8]InstructionEncoding

func GetInstLookupTable() InstLookupTable {
	lookupTable := InstLookupTable{}

	for _, enc := range InstructionTable8086.Encodings {
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

func getVariations(instructionBits []InstructionBits) ([]uint8, error) {
	return _getVariations(instructionBits, 0, 0)
}

func _getVariations(instructionBits []InstructionBits, variationPrefix uint8, bitsSoFar int) ([]uint8, error) {
	if len(instructionBits) == 0 {
		return []uint8{variationPrefix}, nil
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
		variations := []uint8{}
		count := uint8(math.Pow(2, float64(instBit.BitCount)))
		for i := uint8(0); i < count; i++ {
			nextPrefix := variationPrefix<<instBit.BitCount | i
			_variations, error := _getVariations(instructionBits[1:], nextPrefix, bitsSoFar)
			if error != nil {
				return nil, error
			}

			variations = append(variations, _variations...)
		}

		return variations, nil
	}

	return []uint8{}, nil
}
