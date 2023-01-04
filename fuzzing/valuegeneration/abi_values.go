package valuegeneration

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/utils/reflectionutils"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

// addressJSONContractNameOverridePrefix defines a string prefix which is to be followed by a contract name. The
// contract address will be resolved by searching the deployed contracts for a contract with this name.
const addressJSONContractNameOverridePrefix = "DeployedContract:"

// GenerateAbiValue generates a value of the provided abi.Type using the provided ValueGenerator.
// The generated value is returned.
func GenerateAbiValue(generator ValueGenerator, inputType *abi.Type) any {
	// Determine the type of value to generate based on the ABI type.
	if inputType.T == abi.AddressTy {
		return generator.GenerateAddress()
	} else if inputType.T == abi.UintTy {
		if inputType.Size == 64 {
			return generator.GenerateInteger(false, inputType.Size).Uint64()
		} else if inputType.Size == 32 {
			return uint32(generator.GenerateInteger(false, inputType.Size).Uint64())
		} else if inputType.Size == 16 {
			return uint16(generator.GenerateInteger(false, inputType.Size).Uint64())
		} else if inputType.Size == 8 {
			return uint8(generator.GenerateInteger(false, inputType.Size).Uint64())
		} else {
			return generator.GenerateInteger(false, inputType.Size)
		}
	} else if inputType.T == abi.IntTy {
		if inputType.Size == 64 {
			return generator.GenerateInteger(true, inputType.Size).Int64()
		} else if inputType.Size == 32 {
			return int32(generator.GenerateInteger(true, inputType.Size).Int64())
		} else if inputType.Size == 16 {
			return int16(generator.GenerateInteger(true, inputType.Size).Int64())
		} else if inputType.Size == 8 {
			return int8(generator.GenerateInteger(true, inputType.Size).Int64())
		} else {
			return generator.GenerateInteger(true, inputType.Size)
		}
	} else if inputType.T == abi.BoolTy {
		return generator.GenerateBool()
	} else if inputType.T == abi.StringTy {
		return generator.GenerateString()
	} else if inputType.T == abi.BytesTy {
		return generator.GenerateBytes()
	} else if inputType.T == abi.FixedBytesTy {
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		// We opt to keep our API for generators simple, creating the array here and copying elements from a slice.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		bytes := reflect.ValueOf(generator.GenerateFixedBytes(inputType.Size))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(bytes.Index(i))
		}
		return array.Interface()
	} else if inputType.T == abi.ArrayTy {
		// Read notes for fixed bytes to understand the need to create this array through reflection.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < array.Len(); i++ {
			array.Index(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.Elem)))
		}
		return array.Interface()
	} else if inputType.T == abi.SliceTy {
		// Dynamic sized arrays are represented as slices.
		sliceSize := generator.GenerateArrayLength()
		slice := reflect.MakeSlice(inputType.GetType(), sliceSize, sliceSize)
		for i := 0; i < slice.Len(); i++ {
			slice.Index(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.Elem)))
		}
		return slice.Interface()
	} else if inputType.T == abi.TupleTy {
		// Tuples are used to represent structs. For go-ethereum's ABI provider, we're intended to supply matching
		// struct implementations, so we create and populate them through reflection.
		st := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < len(inputType.TupleElems); i++ {
			st.Field(i).Set(reflect.ValueOf(GenerateAbiValue(generator, inputType.TupleElems[i])))
		}
		return st.Interface()
	}

	// Unexpected types will result in a panic as we should support these values as soon as possible:
	// - Mappings cannot be used in public/external methods and must reference storage, so we shouldn't ever
	//	 see cases of it unless Solidity was updated in the future.
	// - FixedPoint types are currently unsupported.
	panic(fmt.Sprintf("attempt to generate function argument of unsupported type: '%s'", inputType.String()))
}

// EncodeJSONArgumentsToMap encodes provided go-ethereum ABI packable input values into a generic JSON type values
// (e.g. []any, map[string]any, etc). It returns the encoded values, or an error if one occurs.
func EncodeJSONArgumentsToMap(inputs abi.Arguments, values []any) (map[string]any, error) {
	// Create a variable to store encoded arguments, fill it with the respective encoded arguments.
	var encodedArgs = make(map[string]any)
	for i, input := range inputs {
		arg, err := encodeJSONArgument(&input.Type, values[i])
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return nil, err
		}
		encodedArgs[input.Name] = arg
	}
	return encodedArgs, nil
}

// EncodeJSONArgumentsToSlice encodes provided go-ethereum ABI packable input values into a generic JSON type values
// (e.g. []any, map[string]any, etc). It returns the encoded values, or an error if one occurs.
func EncodeJSONArgumentsToSlice(inputs abi.Arguments, values []any) ([]any, error) {
	// Create a variable to store encoded arguments, fill it with the respective encoded arguments.
	var encodedArgs = make([]any, len(inputs))
	for i, input := range inputs {
		arg, err := encodeJSONArgument(&input.Type, values[i])
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return nil, err
		}
		encodedArgs[i] = arg
	}
	return encodedArgs, nil
}

// encodeJSONArgument encodes a provided go-ethereum ABI packable input value of a given type, into a generic JSON type
// (e.g. []any, map[string]any, etc). It returns the encoded value, or an error if one occurs.
func encodeJSONArgument(inputType *abi.Type, value any) (any, error) {
	switch inputType.T {
	case abi.AddressTy:
		addr, ok := value.(common.Address)
		if !ok {
			return nil, fmt.Errorf("could not encode address input as the value provided is not an address type")
		}
		return addr.String(), nil
	case abi.UintTy:
		switch inputType.Size {
		case 64:
			v, ok := value.(uint64)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(v, 10), nil
		case 32:
			v, ok := value.(uint32)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		case 16:
			v, ok := value.(uint16)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		case 8:
			v, ok := value.(uint8)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatUint(uint64(v), 10), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not encode uint%v input as the value provided is not of the correct type", inputType.Size)
			}
			return v.String(), nil
		}
	case abi.IntTy:
		switch inputType.Size {
		case 64:
			v, ok := value.(int64)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(v, 10), nil
		case 32:
			v, ok := value.(int32)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		case 16:
			v, ok := value.(int16)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		case 8:
			v, ok := value.(int8)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return strconv.FormatInt(int64(v), 10), nil
		default:
			v, ok := value.(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not encode int%v input as the value provided is not of the correct type", inputType.Size)
			}
			return v.String(), nil
		}
	case abi.BoolTy:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("could not encode bool as the value provided is not of the correct type")
		}
		return b, nil
	case abi.StringTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("could not encode string as the value provided is not of the correct type")
		}
		return str, nil
	case abi.BytesTy:
		b, ok := value.([]byte)
		if !ok {
			return nil, fmt.Errorf("could not encode dynamic-sized bytes as the value provided is not of the correct type")
		}
		return hex.EncodeToString(b), nil
	case abi.FixedBytesTy:
		// TODO: Error checking to ensure `value` is of the correct type.
		b := reflectionutils.ArrayToSlice(reflect.ValueOf(value)).([]byte)
		return hex.EncodeToString(b), nil
	case abi.ArrayTy:
		// Encode all underlying elements in our array
		reflectedArray := reflect.ValueOf(value)
		arrayData := make([]any, 0)
		for i := 0; i < reflectedArray.Len(); i++ {
			elementData, err := encodeJSONArgument(inputType.Elem, reflectedArray.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			arrayData = append(arrayData, elementData)
		}
		return arrayData, nil
	case abi.SliceTy:
		// Encode all underlying elements in our slice
		reflectedArray := reflect.ValueOf(value)
		sliceData := make([]any, 0)
		for i := 0; i < reflectedArray.Len(); i++ {
			elementData, err := encodeJSONArgument(inputType.Elem, reflectedArray.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			sliceData = append(sliceData, elementData)
		}
		return sliceData, nil
	case abi.TupleTy:
		// Encode all underlying fields in our tuple/struct.
		reflectedTuple := reflect.ValueOf(value)
		tupleData := make(map[string]any)
		for i := 0; i < len(inputType.TupleElems); i++ {
			fieldData, err := encodeJSONArgument(inputType.TupleElems[i], reflectionutils.GetField(reflectedTuple.FieldByName(inputType.TupleRawNames[i])))
			if err != nil {
				return nil, err
			}
			tupleData[inputType.TupleRawNames[i]] = fieldData
		}
		return tupleData, nil
	default:
		return nil, fmt.Errorf("could not encode argument, type is unsupported: %v", inputType)
	}
}

// DecodeJSONArgumentsFromMap decodes JSON values into a provided values of the given types, or returns an error of one occurs.
// The values provided must be generic JSON types (e.g. []any, map[string]any, etc) which will be transformed into
// a go-ethereum ABI packable values.
func DecodeJSONArgumentsFromMap(inputs abi.Arguments, values map[string]any, deployedContractAddr map[string]common.Address) ([]any, error) {
	// Create a variable to store decoded arguments, fill it with the respective decoded arguments.
	var decodedArgs = make([]any, len(inputs))
	for i, input := range inputs {
		value, ok := values[input.Name]
		if !ok {
			err := fmt.Errorf("constructor argument not provided for: name: %v", input.Name)
			return nil, err
		}
		arg, err := decodeJSONArgument(&input.Type, value, deployedContractAddr)
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, value, err)
			return nil, err
		}
		decodedArgs[i] = arg
	}
	return decodedArgs, nil
}

// DecodeJSONArgumentsFromSlice decodes JSON values into a provided values of the given types, or returns an error of one occurs.
// The values provided must be generic JSON types (e.g. []any, map[string]any, etc) which will be transformed into
// a go-ethereum ABI packable values.
func DecodeJSONArgumentsFromSlice(inputs abi.Arguments, values []any, deployedContractAddr map[string]common.Address) ([]any, error) {
	// Check our argument value count against our ABI method arguments count.
	if len(values) != len(inputs) {
		err := fmt.Errorf("constructor argument count mismatch, expected %v but got %v", len(inputs), len(values))
		return nil, err
	}

	// Create a variable to store decoded arguments, fill it with the respective decoded arguments.
	var decodedArgs = make([]any, len(inputs))
	for i, input := range inputs {
		arg, err := decodeJSONArgument(&input.Type, values[i], deployedContractAddr)
		if err != nil {
			err = fmt.Errorf("ABI value argument could not be decoded from JSON: \n"+
				"name: %v, abi type: %v, value: %v error: %s",
				input.Name, input.Type, values[i], err)
			return nil, err
		}
		decodedArgs[i] = arg
	}
	return decodedArgs, nil
}

// decodeJSONArgument decodes JSON value into a provided value of a given type, or returns an error of one occurs.
// The value provided must be a generic JSON type (e.g. []any, map[string]any, etc) which will be transformed into
// a go-ethereum ABI packable value.
func decodeJSONArgument(inputType *abi.Type, value any, deployedContractAddr map[string]common.Address) (any, error) {
	var v any
	switch inputType.T {
	case abi.AddressTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("address value should be added as string in JSON")
		}
		// Check if this is a Magic value to get deployed contract address
		if _, contractName, found := strings.Cut(str, addressJSONContractNameOverridePrefix); found {
			v, ok = deployedContractAddr[contractName]
			if !ok {
				return nil, fmt.Errorf("contract %s not found in deployed contracts", contractName)
			}
		} else {
			if !((len(str) == (common.AddressLength*2 + 2)) || (len(str) == common.AddressLength*2)) {
				err := fmt.Errorf("invalid address length (%v)", len(str))
				return nil, err
			}
			v = common.HexToAddress(str)
		}
	case abi.UintTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("integer value should be specified as a string in JSON")
		}
		val := big.NewInt(0)
		_, success := val.SetString(str, 0)
		if !success {
			return nil, fmt.Errorf("invalid integer value")
		}
		switch inputType.Size {
		case 64:
			v = val.Uint64()
		case 32:
			v = uint32(val.Uint64())
		case 16:
			v = uint16(val.Uint64())
		case 8:
			v = uint8(val.Uint64())
		default:
			v = val
		}
	case abi.IntTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("integer value should be added as a string in JSON")
		}
		val := big.NewInt(0)
		_, success := val.SetString(str, 0)
		if !success {
			return nil, fmt.Errorf("invalid integer value")
		}
		switch inputType.Size {
		case 64:
			v = val.Int64()
		case 32:
			v = int32(val.Int64())
		case 16:
			v = int16(val.Int64())
		case 8:
			v = int8(val.Int64())
		default:
			v = val
		}
	case abi.BoolTy:
		str, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid bool value")
		}
		v = str
	case abi.StringTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string value")
		}
		v = str
	case abi.BytesTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("bytes value should be added as string in JSON")
		}
		if len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X') {
			str = str[2:]
		}
		decodedBytes, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}
		v = decodedBytes
	case abi.FixedBytesTy:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s value should be added as string in JSON", inputType)
		}
		if len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X') {
			str = str[2:]
		}
		decodedBytes, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}
		if len(decodedBytes) != inputType.Size {
			return nil, fmt.Errorf("invalid number of bytes %v", len(decodedBytes))
		}

		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		bytesValue := reflect.ValueOf(decodedBytes)
		fixedBytes := reflect.Indirect(reflect.New(inputType.GetType()))
		for i := 0; i < fixedBytes.Len(); i++ {
			fixedBytes.Index(i).Set(bytesValue.Index(i))
		}
		v = fixedBytes.Interface()
	case abi.ArrayTy:
		arr, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid JSON value, array expected")
		}
		// This needs to be an array type, not a slice. But arrays can't be dynamically defined without reflection.
		array := reflect.Indirect(reflect.New(inputType.GetType()))
		for i, e := range arr {
			ele, err := decodeJSONArgument(inputType.Elem, e, deployedContractAddr)
			if err != nil {
				return nil, err
			}
			array.Index(i).Set(reflect.ValueOf(ele))
		}
		v = array.Interface()
	case abi.SliceTy:
		arr, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid JSON value, array expected")
		}
		// Element type of slice is dynamic therefore it needs to be created with reflection.
		slice := reflect.MakeSlice(inputType.GetType(), len(arr), len(arr))
		for i, e := range arr {
			ele, err := decodeJSONArgument(inputType.Elem, e, deployedContractAddr)
			if err != nil {
				return nil, err
			}
			slice.Index(i).Set(reflect.ValueOf(ele))
		}
		v = slice.Interface()
	case abi.TupleTy:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid JSON value, object expected")
		}
		// Tuples are used to represent structs. struct fields are dynamic therefore we create them through reflection.
		st := reflect.Indirect(reflect.New(inputType.GetType()))
		for i, eleType := range inputType.TupleElems {
			fieldName := inputType.TupleRawNames[i]
			fieldValue, ok := object[fieldName]
			if !ok {
				return nil, fmt.Errorf("value for struct field %s not provided", fieldName)
			}
			eleValue, err := decodeJSONArgument(eleType, fieldValue, deployedContractAddr)
			if !ok {
				return nil, fmt.Errorf("can not parse struct field %s, error: %s", fieldName, err)
			}
			st.Field(i).Set(reflect.ValueOf(eleValue))
		}
		v = st.Interface()
	default:
		err := fmt.Errorf("argument type is not supported: %v", inputType)
		return nil, err
	}

	return v, nil
}
