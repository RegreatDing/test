package weightedrandom

import (
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

// Choice describes a weighted, randomly selectable object for use with a Chooser.
type Choice[T any] struct {
	// Data describes the wrapped data that a Chooser should return when making a random Choice selection.
	Data T

	// weight describes a value indicating the likelihood of this Choice to appear in a random selection.
	// Its probability is calculated as current weight / all weights in a Chooser.
	weight *big.Int
}

// NewChoice creates a Choice with the given underlying data and weight to use when added to a Chooser.
func NewChoice[T any](data T, weight *big.Int) *Choice[T] {
	return &Choice[T]{
		Data:   data,
		weight: new(big.Int).Set(weight),
	}
}

// Chooser takes a series of Choice objects which wrap underlying data, and returns one
// of the weighted options randomly.
type Chooser[T any] struct {
	// choices describes the weighted choices from which the chooser will randomly select.
	choices []*Choice[T]

	// totalWeight describes the sum of all weights in choices. This is stored here so it does not need to be
	// recomputed.
	totalWeight *big.Int

	// randomProvider offers a source of random data.
	randomProvider *rand.Rand
	// randomProviderLock is a lock to offer thread safety to the random number generator.
	randomProviderLock *sync.Mutex
}

// NewChooser creates a Chooser with a new random provider and mutex lock.
func NewChooser[T any]() *Chooser[T] {
	return NewChooserWithRand[T](rand.New(rand.NewSource(time.Now().Unix())), &sync.Mutex{})
}

// NewChooserWithRand creates a Chooser with the provided random provider and mutex lock to be acquired when using it.
func NewChooserWithRand[T any](randomProvider *rand.Rand, randomProviderLock *sync.Mutex) *Chooser[T] {
	return &Chooser[T]{
		choices:            make([]*Choice[T], 0),
		randomProvider:     randomProvider,
		randomProviderLock: randomProviderLock,
	}
}

// AddChoices adds weighted choices to the Chooser, allowing for future random selection.
func (c *Chooser[T]) AddChoices(choices ...*Choice[T]) {
	// Acquire our lock during the duration of this method.
	c.randomProviderLock.Lock()
	defer c.randomProviderLock.Unlock()

	// Loop for each choice to add to sum all weights
	for _, choice := range choices {
		c.totalWeight = new(big.Int).Add(c.totalWeight, choice.weight)
	}

	// Add to choices to our array
	c.choices = append(c.choices, choices...)
}

// Choose selects a random weighted item from the Chooser, or returns an error if one occurs.
func (c *Chooser[T]) Choose() (*T, error) {
	// If we have no choices or 0 total weight, return nil.
	if len(c.choices) == 0 || c.totalWeight.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("could not return a weighted random choice because no choices exist with non-zero weights")
	}

	// Acquire our lock during the duration of this method.
	c.randomProviderLock.Lock()
	defer c.randomProviderLock.Unlock()

	// Next we'll determine how many bits/bytes are needed to represent our random value
	bitLength := c.totalWeight.BitLen()
	byteLength := bitLength / 8
	unusedBits := bitLength % 8
	if unusedBits != 0 {
		byteLength += 1
	}

	// Generate the number of bytes needed.
	randomData := make([]byte, c.totalWeight.BitLen())
	_, err := c.randomProvider.Read(randomData)
	if err != nil {
		return nil, err
	}

	// If we have unused bits, we'll want to mask/clear them out (big.Int uses big endian for byte parsing)
	randomData[0] = randomData[0] & (byte(0xFF) >> unusedBits)

	// We use these bytes to get an index in [0, total weight] to use to return an item.
	// TODO: this may be the correct bit size but have too many bits set to actually be in range, so we perform
	//  modulus division to wrap around. This isn't fully uniform in distribution, we should consider revisiting this.
	selectedWeightPosition := new(big.Int).SetBytes(randomData)
	selectedWeightPosition = new(big.Int).Mod(selectedWeightPosition, c.totalWeight)

	// Loop for each item
	for _, choice := range c.choices {

		// If our selected weight position is in range for this item, return it
		if selectedWeightPosition.Cmp(choice.weight) < 0 {
			return &choice.Data, nil
		}

		// Subtract the choice weight from the current position, and go to the next item to see if it's in range.
		selectedWeightPosition = new(big.Int).Sub(selectedWeightPosition, choice.weight)
	}

	return nil, fmt.Errorf("could not obtain a weighted random choice, selected position does not exist")
}
