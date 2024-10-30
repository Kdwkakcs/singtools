package parse

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/sagernet/sing-box/experimental/libbox/platform"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
)

type OptionsEntry struct {
	content []byte
	path    string
	options option.Options
}

func readConfigAt(path string) (*OptionsEntry, error) {
	var (
		configContent []byte
		err           error
	)
	configContent, err = os.ReadFile(path)
	if err != nil {
		return nil, E.Cause(err, "read config at ", path)
	}
	options, err := json.UnmarshalExtended[option.Options](configContent)
	if err != nil {
		return nil, E.Cause(err, "decode config at ", path)
	}
	return &OptionsEntry{
		content: configContent,
		path:    path,
		options: options,
	}, nil
}

func readConfig(configPaths []string) ([]*OptionsEntry, error) {
	var optionsList []*OptionsEntry
	for _, path := range configPaths {
		optionsEntry, err := readConfigAt(path)
		if err != nil {
			return nil, err
		}
		optionsList = append(optionsList, optionsEntry)
	}
	sort.Slice(optionsList, func(i, j int) bool {
		return optionsList[i].path < optionsList[j].path
	})
	return optionsList, nil
}

func readConfigAndMerge(configPaths []string) (option.Options, error) {
	optionsList, err := readConfig(configPaths)
	if err != nil {
		return option.Options{}, err
	}
	if len(optionsList) == 1 {
		return optionsList[0].options, nil
	}
	var mergedMessage json.RawMessage
	for _, options := range optionsList {
		mergedMessage, err = badjson.MergeJSON(options.options.RawMessage, mergedMessage, false)
		if err != nil {
			return option.Options{}, E.Cause(err, "merge config at ", options.path)
		}
	}
	var mergedOptions option.Options
	err = mergedOptions.UnmarshalJSON(mergedMessage)
	if err != nil {
		return option.Options{}, E.Cause(err, "unmarshal merged config")
	}
	return mergedOptions, nil
}

// add a function to remove the router in the json config
func remove_other(options option.Options) option.Options {
	options.DNS = nil
	options.Route = nil
	return options
}

type BoxOptions struct {
	option.Options
	Context           context.Context
	PlatformInterface platform.Interface
	PlatformLogWriter log.PlatformWriter
}

func ReadConfigs(configPaths []string) (BoxOptions, error) {
	// check the all paths are exists
	for _, path := range configPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return BoxOptions{}, E.New("config file not found at ", path)
		}
	}
	// read and merge configs
	options, err := readConfigAndMerge(configPaths)
	options = remove_other(options)
	fmt.Printf("%+v\n", options.DNS)

	if err != nil {
		return BoxOptions{}, err
	}
	// create box options
	boxOptions := BoxOptions{
		Options: options,
		Context: context.Background(),
	}
	return boxOptions, nil
}
