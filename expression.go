package main

import (
	"fmt"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// GetElementPtr translates a getelementptr expression.
func GetElementPtr(elemType types.Type, src value.Value, indices []value.Value) (string, error) {
	srcPointerType, ok := src.Type().(*types.PointerType)
	if !ok {
		return "", fmt.Errorf("non-pointer source parameter: %v", src.Type())
	}
	if !types.Equal(srcPointerType.ElemType, elemType) {
		return "", fmt.Errorf("mismatched source and element types")
	}

	zeroFirstIndex := false
	positiveFirstIndex := false
	if firstIndex, ok := indices[0].(*constant.Int); ok {
		switch firstIndex.X.Sign() {
		case 0:
			zeroFirstIndex = true
		case 1:
			positiveFirstIndex = true
		}
	}
	takeAddress := false

	source, err := FormatValue(src)
	if err != nil {
		return "", fmt.Errorf("error translating source pointer (%q): %v", src, err)
	}
	result := source

	if !zeroFirstIndex {
		firstIndex, err := FormatValue(indices[0])
		if err != nil {
			return "", fmt.Errorf("error translating first index (%v): %v", indices[0], err)
		}
		et, err := TypeSpec(elemType)
		if err != nil {
			return "", fmt.Errorf("error translating element type (%v): %v", elemType, err)
		}
		offset := fmt.Sprintf("uintptr(int64(%s)) * unsafe.Sizeof(*(*%s)(nil))", firstIndex, et)
		if positiveFirstIndex {
			offset = fmt.Sprintf("%s * unsafe.Sizeof(*(*%s)(nil))", firstIndex, et)
		}
		result = fmt.Sprintf("uintptr(unsafe.Pointer(%s)) + %s", source, offset)
		result = fmt.Sprintf("(*%s)(unsafe.Pointer(%s))", et, result)
	}

	currentType := elemType

	for _, index := range indices[1:] {
		switch ct := currentType.(type) {
		case *types.ArrayType:
			v, err := FormatValue(index)
			if err != nil {
				return "", fmt.Errorf("error translating index (%v): %v", index, err)
			}
			result = fmt.Sprintf("%s[%s]", result, v)
			currentType = ct.ElemType
			takeAddress = true

		case *types.StructType:
			ci, ok := index.(*constant.Int)
			if !ok {
				return "", fmt.Errorf("non-constant index into struct: %v", index)
			}
			result = fmt.Sprintf("%s.f%v", result, ci.X)
			currentType = ct.Fields[ci.X.Int64()]
			takeAddress = true

		default:
			return "", fmt.Errorf("unsupported type to index into: %v", currentType)
		}
	}

	if takeAddress {
		result = "&" + result
	}

	return result, nil
}
