package valuegeneration

import (
	"math/big"
	"math/rand"

	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common"
)

// MutatingValueGenerator is a provider used to generate function inputs and call arguments using mutation-based
// approaches against items within a base_value_set.ValueSet, such as AST literals.
type ShrinkingValueGenerator struct {
	// ValueSet contains a set of values which the ValueGenerator may use to aid in value generation and mutation
	// operations.
	valueSet *ValueSet

	// RandomValueGenerator is included to inherit from the random generator
	*RandomValueGenerator
}

// NewMutatingValueGenerator creates a new MutatingValueGenerator using a provided base_value_set.ValueSet to seed base-values for mutation.
func NewShrinkingValueGenerator(valueSet *ValueSet, randomProvider *rand.Rand) *ShrinkingValueGenerator {
	// Create and return our generator
	generator := &ShrinkingValueGenerator{
		valueSet:             valueSet,
		RandomValueGenerator: NewRandomValueGenerator(nil, randomProvider),
	}

	// Ensure some initial values this mutator will depend on for basic mutations to the set.
	generator.valueSet.AddInteger(big.NewInt(0))
	generator.valueSet.AddInteger(big.NewInt(1))
	generator.valueSet.AddInteger(big.NewInt(2))
	return generator
}

// integerMutationMethods define methods which take a big integer and a set of inputs and
// transform the integer with a random input and operation. This is used in a loop to create
// mutated integer values.
var integerShrinkingMethods = []func(*ShrinkingValueGenerator, *big.Int, ...*big.Int) *big.Int{
	func(g *ShrinkingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Subtract a random input
		var r *big.Int
		if x.Cmp(big.NewInt(0)) > 0 {
			r = big.NewInt(0).Sub(x, inputs[g.randomProvider.Intn(len(inputs))])
		} else if x.Cmp(big.NewInt(0)) < 0 {
			r = big.NewInt(0).Add(x, inputs[g.randomProvider.Intn(len(inputs))])
		}
		return r

	},
	func(g *ShrinkingValueGenerator, x *big.Int, inputs ...*big.Int) *big.Int {
		// Divide by two
		return big.NewInt(0).Div(x, big.NewInt(2))
	},
}

// mutateIntegerInternal takes an integer input and returns either a random new integer, or a mutated value based off the input.
// If a nil input is provided, this method uses an existing base value set value as the starting point for mutation.
func (g *ShrinkingValueGenerator) shrinkIntegerInternal(i *big.Int, signed bool, bitLength int) *big.Int {
	// Calculate our integer bounds
	min, max := utils.GetIntegerConstraints(signed, bitLength)

	// Obtain our inputs. We also add our min/max values for this range to the list of inputs.
	// Note: We exclude min being added if we're requesting an unsigned integer, as zero is already
	// in our set, and we don't want duplicates.
	var inputs []*big.Int
	inputs = append(inputs, g.valueSet.Integers()...)
	if signed {
		inputs = append(inputs, min, max)
	} else {
		inputs = append(inputs, max)
	}

	// Determine which value we'll use as an initial input, and how many mutations we will perform.
	inputIdx := g.randomProvider.Intn(len(inputs))
	input := new(big.Int)
	if i != nil {
		input.Set(i)
	} else {
		input.Set(inputs[inputIdx])
	}
	input = utils.ConstrainIntegerToBounds(input, min, max)

	// Shrink input
	input = integerShrinkingMethods[g.randomProvider.Intn(len(integerShrinkingMethods))](g, input, inputs...)

	// Correct value boundaries (underflow/overflow)
	input = utils.ConstrainIntegerToBounds(input, min, max)
	return input
}

// MutateInteger takes an integer input and applies optional mutations to the provided value.
// Returns an optionally mutated copy of the input.
func (g *ShrinkingValueGenerator) MutateInteger(i *big.Int, signed bool, bitLength int) *big.Int {
	return g.shrinkIntegerInternal(i, signed, bitLength)
}

// bytesMutationMethods define methods which take an initial bytes and a set of inputs to transform the input. The
// transformed input is returned. This is used in a loop to mutate byte slices.
var bytesShrinkingMethods = []func(*ShrinkingValueGenerator, []byte) []byte{
	// Replace a random index with a random byte
	func(g *ShrinkingValueGenerator, b []byte) []byte {
		// Replace an existing byte in our array with zero.
		if len(b) > 0 {
			b[g.randomProvider.Intn(len(b))] = 0
		}
		return b
	},
	// Remove a random byte
	func(g *ShrinkingValueGenerator, b []byte) []byte {
		// If we have no bytes to remove, do nothing.
		if len(b) == 0 {
			return b
		}

		i := g.randomProvider.Intn(len(b))
		return append(b[:i], b[i+1:]...)
	},
}

// mutateBytesInternal takes a byte array and returns either a random new byte array, or a mutated value based off the
// input.
// If a nil input is provided, this method uses an existing base value set value as the starting point for mutation.
func (g *ShrinkingValueGenerator) shrinkBytesInternal(b []byte) []byte {
	// Mutate the data for our desired number of rounds
	input := bytesShrinkingMethods[g.randomProvider.Intn(len(bytesShrinkingMethods))](g, b)

	return input
}

// stringMutationMethods define methods which take an initial string and a set of inputs to transform the input. The
// transformed input is returned. This is used in a loop to mutate strings.
var shrinkMutationMethods = []func(*ShrinkingValueGenerator, string) string{
	// Replace a random index with a NULL char
	func(g *ShrinkingValueGenerator, s string) string {

		// If the string is empty, we can simply return a new string with just the rune in it.
		r := []rune(s)
		if len(r) == 0 {
			return string(r)
		}

		// Otherwise, we replace a rune in it and return it.
		r[g.randomProvider.Intn(len(r))] = 0
		return string(r)
	},
	// Remove a random character
	func(g *ShrinkingValueGenerator, s string) string {
		// If we have no characters to remove, do nothing
		if len(s) == 0 {
			return s
		}

		// Otherwise, remove a random character.
		i := g.randomProvider.Intn(len(s))
		return s[:i] + s[i+1:]
	},
}

// shrinkString takes a string input and returns a mutated value based off the input.
func (g *ShrinkingValueGenerator) MutateString(s string) string {
	return g.shrinkStringInternal(&s)
}

// mutateStringInternal takes a string and returns either a random new string, or a mutated value based off the input.
// If a nil input is provided, this method uses an existing base value set value as the starting point for mutation.
func (g *ShrinkingValueGenerator) shrinkStringInternal(s *string) string {
	input := shrinkMutationMethods[g.randomProvider.Intn(len(shrinkMutationMethods))](g, *s)

	return input
}

// MutateAddress takes an address input and sometimes returns a mutated value based off the input.
func (g *ShrinkingValueGenerator) MutateAddress(addr common.Address) common.Address {
	addressBytes := make([]byte, common.AddressLength)
	return common.BytesToAddress(addressBytes)
}
