package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/crytic/medusa/chain/config"
	"os"

	"github.com/crytic/medusa/compilation"
	"github.com/crytic/medusa/utils"
)

type ProjectConfig struct {
	// Fuzzing describes the configuration used in fuzzing campaigns.
	Fuzzing FuzzingConfig `json:"fuzzing"`

	// Compilation describes the configuration used to compile the underlying project.
	Compilation *compilation.CompilationConfig `json:"compilation"`
}

// FuzzingConfig describes the configuration options used by the fuzzing.Fuzzer.
type FuzzingConfig struct {
	// Workers describes the amount of threads to use in fuzzing campaigns.
	Workers int `json:"workers"`

	// WorkerResetLimit describes how many call sequences a worker should test before it is destroyed and recreated
	// so that memory from its underlying chain is freed.
	WorkerResetLimit int `json:"workerResetLimit"`

	// Timeout describes a time in seconds for which the fuzzing operation should run. Providing negative or zero value
	// will result in no timeout.
	Timeout int `json:"timeout"`

	// TestLimit describes a threshold for the number of transactions to test, after which it will exit. This number
	// must be non-negative. A zero value indicates the test limit should not be enforced.
	TestLimit uint64 `json:"testLimit"`

	// CallSequenceLength describes the maximum length a transaction sequence can be generated as.
	CallSequenceLength int `json:"callSequenceLength"`

	// CorpusDirectory describes the name for the folder that will hold the corpus and the coverage files. If empty,
	// the in-memory corpus will be used, but not flush to disk.
	CorpusDirectory string `json:"corpusDirectory"`

	// CoverageEnabled describes whether to use coverage-guided fuzzing
	CoverageEnabled bool `json:"coverageEnabled"`

	// DeploymentOrder determines the order in which the contracts should be deployed
	DeploymentOrder []string `json:"deploymentOrder"`

	// Constructor arguments for contracts deployment. It is available only in init mode
	ConstructorArgs map[string]map[string]any `json:"constructorArgs"`

	// DeployerAddress describe the account address to be used to deploy contracts.
	DeployerAddress string `json:"deployerAddress"`

	// SenderAddresses describe a set of account addresses to be used to send state-changing txs (calls) in fuzzing
	// campaigns.
	SenderAddresses []string `json:"senderAddresses"`

	// MaxBlockNumberDelay describes the maximum distance in block numbers the fuzzer will use when generating blocks
	// compared to the previous.
	MaxBlockNumberDelay uint64 `json:"blockNumberDelayMax"`

	// MaxBlockTimestampDelay describes the maximum distance in timestamps the fuzzer will use when generating blocks
	// compared to the previous.
	MaxBlockTimestampDelay uint64 `json:"blockTimestampDelayMax"`

	// BlockGasLimit describes the maximum amount of gas that can be used in a block by transactions. This defines
	// limits for how many transactions can be included per block.
	BlockGasLimit uint64 `json:"blockGasLimit"`

	// TransactionGasLimit describes the maximum amount of gas that will be used by the fuzzer generated transactions.
	TransactionGasLimit uint64 `json:"transactionGasLimit"`

	// Testing describes the configuration used for different testing strategies.
	Testing TestingConfig `json:"testing"`

	// TestChainConfig represents the chain.TestChain config to use when initializing a chain.
	TestChainConfig config.TestChainConfig `json:"chainConfig"`
}

// TestingConfig describes the configuration options used for testing
type TestingConfig struct {
	// StopOnFailedTest describes whether the fuzzing.Fuzzer should stop after detecting the first failed test.
	StopOnFailedTest bool `json:"stopOnFailedTest"`

	// StopOnFailedContractMatching describes whether the fuzzing.Fuzzer should stop after failing to match bytecode
	// to determine which contract a deployed contract is.
	StopOnFailedContractMatching bool `json:"stopOnFailedContractMatching"`

	// TestAllContracts indicates whether all contracts should be tested (including dynamically deployed ones), rather
	// than just the contracts specified in the project configuration's deployment order.
	TestAllContracts bool `json:"testAllContracts"`

	// TraceAll describes whether a trace should be attached to each element of a finalized shrunken call sequence,
	// e.g. when a call sequence triggers a test failure. Test providers may attach execution traces by default,
	// even if this option is not enabled.
	TraceAll bool `json:"traceAll"`

	// AssertionTesting describes the configuration used for assertion testing.
	AssertionTesting AssertionTestingConfig `json:"assertionTesting"`

	// PropertyTesting describes the configuration used for property testing.
	PropertyTesting PropertyTestConfig `json:"propertyTesting"`
}

// AssertionTestingConfig describes the configuration options used for assertion testing
type AssertionTestingConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestViewMethods dictates whether constant/pure/view methods should be tested.
	TestViewMethods bool `json:"testViewMethods"`
}

// PropertyTestConfig describes the configuration options used for property testing
type PropertyTestConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestPrefixes dictates what method name prefixes will determine if a contract method is a property test.
	TestPrefixes []string `json:"testPrefixes"`
}

// ReadProjectConfigFromFile reads a JSON-serialized ProjectConfig from a provided file path.
// Returns the ProjectConfig if it succeeds, or an error if one occurs.
func ReadProjectConfigFromFile(path string) (*ProjectConfig, error) {
	// Read our project configuration file data
	fmt.Printf("Reading configuration file: %s\n", path)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the project configuration
	projectConfig, err := GetDefaultProjectConfig("")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, projectConfig)
	if err != nil {
		return nil, err
	}

	return projectConfig, nil
}

// WriteToFile writes the ProjectConfig to a provided file path in a JSON-serialized format.
// Returns an error if one occurs.
func (p *ProjectConfig) WriteToFile(path string) error {
	// Serialize the configuration
	b, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return err
	}

	// Save it to the provided output path and return the result
	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Validate validates that the ProjectConfig meets certain requirements.
// Returns an error if one occurs.
func (p *ProjectConfig) Validate() error {
	// Verify the worker count is a positive number.
	if p.Fuzzing.Workers <= 0 {
		return errors.New("project configuration must specify a positive number for the worker count")
	}

	// Verify that the sequence length is a positive number
	if p.Fuzzing.CallSequenceLength <= 0 {
		return errors.New("project configuration must specify a positive number for the transaction sequence length")
	}

	// Verify the worker reset limit is a positive number
	if p.Fuzzing.WorkerResetLimit <= 0 {
		return errors.New("project configuration must specify a positive number for the worker reset limit")
	}

	// Verify gas limits are appropriate
	if p.Fuzzing.BlockGasLimit < p.Fuzzing.TransactionGasLimit {
		return errors.New("project configuration must specify a block gas limit which is not less than the transaction gas limit")
	}
	if p.Fuzzing.BlockGasLimit == 0 || p.Fuzzing.TransactionGasLimit == 0 {
		return errors.New("project configuration must specify a block and transaction gas limit which is non-zero")
	}

	// Verify that senders are well-formed addresses
	if _, err := utils.HexStringsToAddresses(p.Fuzzing.SenderAddresses); err != nil {
		return errors.New("project configuration must specify only well-formed sender address(es)")
	}

	// Verify that deployer is a well-formed address
	if _, err := utils.HexStringToAddress(p.Fuzzing.DeployerAddress); err != nil {
		return errors.New("project configuration must specify only a well-formed deployer address")
	}

	// Verify property testing fields.
	if p.Fuzzing.Testing.PropertyTesting.Enabled {
		// Test prefixes must be supplied if property testing is enabled.
		if len(p.Fuzzing.Testing.PropertyTesting.TestPrefixes) == 0 {
			return errors.New("project configuration must specify test name prefixes if property testing is enabled")
		}
	}
	return nil
}
