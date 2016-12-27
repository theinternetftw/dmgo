package dmgo

import (
	"bytes"
	"io"
)

// gobi - go, batteries included.
//
// embeddable single file version

var gobi = struct {
	// Int provides access to int-related functions.
	Int gobiInt
	// Int8 provides access to int8-related functions.
	Int8 gobiInt8
	// Int16 provides access to int16-related functions.
	Int16 gobiInt16
	// Int32 provides access to int32-related functions.
	Int32 gobiInt32
	// Rune provides access to rune-related functions.
	Rune gobiRune
	// Uint provides access to uint-related functions.
	Uint gobiUint
	// Uint16 provides access to uint16-related functions.
	Uint16 gobiUint16
	// Uint32 provides access to uint32-related functions.
	Uint32 gobiUint32
	// Byte provides access to byte-related functions.
	Byte gobiByte
}{}

type gobiInt struct {
	// Slice provides access to functions on slices of ints.
	Slice gobiIntSlice
}
type gobiIntSlice struct{}
type gobiInt8 struct {
	// Slice provides access to functions on slices of int8s.
	Slice gobiInt8Slice
}
type gobiInt8Slice struct{}
type gobiInt16 struct {
	// Slice provides access to functions on slices of int16s.
	Slice gobiInt16Slice
}
type gobiInt16Slice struct{}
type gobiInt32 struct {
	// Slice provides access to functions on slices of int32s.
	Slice gobiInt32Slice
}
type gobiInt32Slice struct{}
type gobiRune struct {
	// Slice provides access to functions on slices of runes.
	Slice gobiRuneSlice
}
type gobiRuneSlice struct{}
type gobiUint struct {
	// Slice provides access to functions on slices of uints.
	Slice gobiUintSlice
}
type gobiUintSlice struct{}
type gobiUint16 struct {
	// Slice provides access to functions on slices of uint16s.
	Slice gobiUint16Slice
}
type gobiUint16Slice struct{}
type gobiUint32 struct {
	// Slice provides access to functions on slices of uint32s.
	Slice gobiUint32Slice
}
type gobiUint32Slice struct{}
type gobiByte struct {
	// Slice provides access to functions on slices of bytes.
	Slice gobiByteSlice
}
type gobiByteSlice struct{}

// Max returns the maximum of one or more ints
func (u gobiInt) Max(num1 int, nums ...int) int {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more ints
func (u gobiInt) Min(num1 int, nums ...int) int {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of ints
func (u gobiInt) Sum(nums ...int) int {
	sum := int(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more int8s
func (u gobiInt8) Max(num1 int8, nums ...int8) int8 {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more int8s
func (u gobiInt8) Min(num1 int8, nums ...int8) int8 {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of int8s
func (u gobiInt8) Sum(nums ...int8) int8 {
	sum := int8(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more int16s
func (u gobiInt16) Max(num1 int16, nums ...int16) int16 {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more int16s
func (u gobiInt16) Min(num1 int16, nums ...int16) int16 {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of int16s
func (u gobiInt16) Sum(nums ...int16) int16 {
	sum := int16(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more int32s
func (u gobiInt32) Max(num1 int32, nums ...int32) int32 {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more int32s
func (u gobiInt32) Min(num1 int32, nums ...int32) int32 {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of int32s
func (u gobiInt32) Sum(nums ...int32) int32 {
	sum := int32(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more runes
func (u gobiRune) Max(num1 rune, nums ...rune) rune {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more runes
func (u gobiRune) Min(num1 rune, nums ...rune) rune {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of runes
func (u gobiRune) Sum(nums ...rune) rune {
	sum := rune(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more uints
func (u gobiUint) Max(num1 uint, nums ...uint) uint {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more uints
func (u gobiUint) Min(num1 uint, nums ...uint) uint {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of uints
func (u gobiUint) Sum(nums ...uint) uint {
	sum := uint(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more uint16s
func (u gobiUint16) Max(num1 uint16, nums ...uint16) uint16 {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more uint16s
func (u gobiUint16) Min(num1 uint16, nums ...uint16) uint16 {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of uint16s
func (u gobiUint16) Sum(nums ...uint16) uint16 {
	sum := uint16(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more uint32s
func (u gobiUint32) Max(num1 uint32, nums ...uint32) uint32 {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more uint32s
func (u gobiUint32) Min(num1 uint32, nums ...uint32) uint32 {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of uint32s
func (u gobiUint32) Sum(nums ...uint32) uint32 {
	sum := uint32(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Max returns the maximum of one or more bytes
func (u gobiByte) Max(num1 byte, nums ...byte) byte {
	max := num1
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

// Min returns the minimum of one or more bytes
func (u gobiByte) Min(num1 byte, nums ...byte) byte {
	min := num1
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return min
}

// Sum returns the sum of any number of bytes
func (u gobiByte) Sum(nums ...byte) byte {
	sum := byte(0)
	for _, num := range nums {
		sum += num
	}
	return sum
}

// AppendZeroes adds some number of zeroes to the end of a slice of ints.
func (u gobiIntSlice) AppendZeroes(s []int, numZeroes int) []int {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of ints.
func (u gobiIntSlice) PrependZeroes(s []int, numZeroes int) []int {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of int8s.
func (u gobiInt8Slice) AppendZeroes(s []int8, numZeroes int) []int8 {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int8, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of int8s.
func (u gobiInt8Slice) PrependZeroes(s []int8, numZeroes int) []int8 {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int8, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of int16s.
func (u gobiInt16Slice) AppendZeroes(s []int16, numZeroes int) []int16 {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int16, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of int16s.
func (u gobiInt16Slice) PrependZeroes(s []int16, numZeroes int) []int16 {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int16, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of int32s.
func (u gobiInt32Slice) AppendZeroes(s []int32, numZeroes int) []int32 {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int32, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of int32s.
func (u gobiInt32Slice) PrependZeroes(s []int32, numZeroes int) []int32 {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]int32, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of runes.
func (u gobiRuneSlice) AppendZeroes(s []rune, numZeroes int) []rune {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]rune, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of runes.
func (u gobiRuneSlice) PrependZeroes(s []rune, numZeroes int) []rune {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]rune, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of uints.
func (u gobiUintSlice) AppendZeroes(s []uint, numZeroes int) []uint {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]uint, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of uints.
func (u gobiUintSlice) PrependZeroes(s []uint, numZeroes int) []uint {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]uint, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of uint16s.
func (u gobiUint16Slice) AppendZeroes(s []uint16, numZeroes int) []uint16 {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]uint16, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of uint16s.
func (u gobiUint16Slice) PrependZeroes(s []uint16, numZeroes int) []uint16 {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]uint16, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of uint32s.
func (u gobiUint32Slice) AppendZeroes(s []uint32, numZeroes int) []uint32 {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]uint32, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of uint32s.
func (u gobiUint32Slice) PrependZeroes(s []uint32, numZeroes int) []uint32 {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]uint32, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the end of a slice of bytes.
func (u gobiByteSlice) AppendZeroes(s []byte, numZeroes int) []byte {
	if numZeroes < 0 {
		panic("AppendZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]byte, finalLen)
		copy(newSlice, s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	for i := startLen; i < finalLen; i++ {
		s[i] = 0
	}
	return s
}

// AppendZeroes adds some number of zeroes to the beginning of a slice of bytes.
func (u gobiByteSlice) PrependZeroes(s []byte, numZeroes int) []byte {
	if numZeroes < 0 {
		panic("PrependZeroes: negative number of zeroes requested")
	}
	finalLen := len(s) + numZeroes
	if cap(s) < finalLen {
		newSlice := make([]byte, finalLen)
		copy(newSlice[numZeroes:], s)
		return newSlice
	}
	startLen := len(s)
	s = s[:finalLen]
	copy(s[startLen:], s[:startLen])
	for i := 0; i < startLen; i++ {
		s[i] = 0
	}
	return s
}

// SaturatedAdd performs a saturated add on one or more uints.
func (u gobiUint) SaturatedAdd(num1 uint, nums ...uint) uint {
	sum := num1
	for _, num := range nums {
		newSum := sum + num
		if newSum < sum || newSum < num {
			return ^uint(0)
		}
		sum = newSum
	}
	return sum
}

// SaturatedAdd performs a saturated add on one or more uint16s.
func (u gobiUint16) SaturatedAdd(num1 uint16, nums ...uint16) uint16 {
	sum := num1
	for _, num := range nums {
		newSum := sum + num
		if newSum < sum || newSum < num {
			return ^uint16(0)
		}
		sum = newSum
	}
	return sum
}

// SaturatedAdd performs a saturated add on one or more uint32s.
func (u gobiUint32) SaturatedAdd(num1 uint32, nums ...uint32) uint32 {
	sum := num1
	for _, num := range nums {
		newSum := sum + num
		if newSum < sum || newSum < num {
			return ^uint32(0)
		}
		sum = newSum
	}
	return sum
}

// SaturatedAdd performs a saturated add on one or more bytes.
func (u gobiByte) SaturatedAdd(num1 byte, nums ...byte) byte {
	sum := num1
	for _, num := range nums {
		newSum := sum + num
		if newSum < sum || newSum < num {
			return ^byte(0)
		}
		sum = newSum
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiIntSlice) Cat(slices ...[]int) []int {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]int, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiIntSlice) TotalLen(slices ...[]int) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiInt8Slice) Cat(slices ...[]int8) []int8 {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]int8, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiInt8Slice) TotalLen(slices ...[]int8) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiInt16Slice) Cat(slices ...[]int16) []int16 {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]int16, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiInt16Slice) TotalLen(slices ...[]int16) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiInt32Slice) Cat(slices ...[]int32) []int32 {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]int32, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiInt32Slice) TotalLen(slices ...[]int32) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiRuneSlice) Cat(slices ...[]rune) []rune {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]rune, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiRuneSlice) TotalLen(slices ...[]rune) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiUintSlice) Cat(slices ...[]uint) []uint {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]uint, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiUintSlice) TotalLen(slices ...[]uint) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiUint16Slice) Cat(slices ...[]uint16) []uint16 {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]uint16, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiUint16Slice) TotalLen(slices ...[]uint16) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiUint32Slice) Cat(slices ...[]uint32) []uint32 {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]uint32, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiUint32Slice) TotalLen(slices ...[]uint32) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Cat concatenates zero or more slices.
func (u gobiByteSlice) Cat(slices ...[]byte) []byte {
	size := 0
	for i := range slices {
		size += len(slices[i])
	}
	result := make([]byte, size)[:0]
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// TotalLen returns the total length of all provided slices.
func (u gobiByteSlice) TotalLen(slices ...[]byte) int {
	sum := 0
	for _, s := range slices {
		sum += len(s)
	}
	return sum
}

// Slice functions for bytes only
// NOTE: this section not generated

// MultiReader allows you to easily create a single reader from
// a number of byte slices.
func (u gobiByteSlice) MultiReader(slices ...[]byte) io.Reader {
	readers := make([]io.Reader, len(slices))
	for i := range slices {
		readers[i] = bytes.NewReader(slices[i])
	}
	return io.MultiReader(readers...)
}

// Read allows you to read the contents of an io.Reader into
// any number of pre-sized byte slices.
//
// Example usage:
// hdr1 := make([]byte, 0x18)
// hdr2 := make([]byte, 0x30)
// data := make([]byte, 0x100)
// numBytes, err := gobi.Byte.Slice.Read(netReader, hdr1, hdr2, data)
//
// This function is for where using multiple slices would be convenient.
// Use encoding/binary for when you're ultimately moving things into
// well-defined structs.
//
func (u gobiByteSlice) Read(r io.Reader, slices ...[]byte) error {
	for _, s := range slices {
		if _, err := io.ReadFull(r, s); err != nil {
			return err
		}
	}
	return nil
}

// Write allows you to write the contents of any number of byte slices
// to a single io.Writer. It returns an error if any slice could not be written.
func (u gobiByteSlice) Write(w io.Writer, slices ...[]byte) error {
	_, err := io.Copy(w, gobi.Byte.Slice.MultiReader(slices...))
	return err
}
