/*
 * Copyright 2019-present Ciena Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package commands

import (
	"context"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/opencord/voltctl/pkg/format"
	"github.com/opencord/voltctl/pkg/model"
	"github.com/opencord/voltha-lib-go/v3/pkg/config"
	"github.com/opencord/voltha-lib-go/v3/pkg/log"
	"strings"
)

const (
	defaultComponentName = "global"
	defaultPackageName   = "default"
)

// LogLevelOutput represents the  output structure for the loglevel
type LogLevelOutput struct {
	ComponentName string
	Status        string
	Error         string
}

// SetLogLevelOpts represents the given input for the set loglevel
type SetLogLevelOpts struct {
	OutputOptions
	Args struct {
		Level     string
		Component []string
	} `positional-args:"yes" required:"yes"`
}

// ListLogLevelOpts represents the given input for the list loglevel
type ListLogLevelsOpts struct {
	ListOutputOptions
	Args struct {
		Component []string
	} `positional-args:"yes" required:"yes"`
}

// ClearLogLevelOpts represents the given input for the clear loglevel
type ClearLogLevelsOpts struct {
	OutputOptions
	Args struct {
		Component []string
	} `positional-args:"yes" required:"yes"`
}

// LogLevelOpts represents the loglevel commands
type LogLevelOpts struct {
	SetLogLevel    SetLogLevelOpts    `command:"set"`
	ListLogLevels  ListLogLevelsOpts  `command:"list"`
	ClearLogLevels ClearLogLevelsOpts `command:"clear"`
}

var logLevelOpts = LogLevelOpts{}

const (
	DEFAULT_LOGLEVELS_FORMAT   = "table{{ .ComponentName }}\t{{.PackageName}}\t{{.Level}}"
	DEFAULT_SETLOGLEVEL_FORMAT = "table{{ .ComponentName }}\t{{.Status}}\t{{.Error}}"
)

// RegisterLogLevelCommands is used to  register set,list and clear loglevel of components
func RegisterLogLevelCommands(parent *flags.Parser) {
	_, err := parent.AddCommand("loglevel", "loglevel commands", "list,set and clear log levels of components", &logLevelOpts)
	if err != nil {
		Error.Fatalf("Unable to register log level commands with voltctl command parser: %s", err.Error())
	}
}

func listGlobalConfig(cConfig *config.ComponentConfig) (string, error) {
	var globalDefaultLogLevel string
	globalLogConfig, err := cConfig.RetrieveAll(context.Background())
	if err != nil {
		return "", err
	}

	if globalLevel, ok := globalLogConfig[defaultPackageName]; ok {
		if _, err := log.StringToLogLevel(globalLevel); err == nil {
			globalDefaultLogLevel = globalLevel
		}
	}

	return globalDefaultLogLevel, nil
}

func listComponentNameConfig(cConfig *config.ComponentConfig, globalLevel string) (map[string]string, error) {

	componentLogConfig, err := cConfig.RetrieveAll(context.Background())
	if err != nil {
		return nil, err
	}

	if globalLevel != "" {
		componentLogConfig[defaultPackageName] = globalLevel
	}
	return componentLogConfig, nil
}

// processCommandArgs stores  the component name and package names given in command arguments to LogLevel
// It checks the given argument has # key or not, if # is present then split the argument for # then stores first part as component name
// and second part as package name
func processCommandArgs(component string) model.LogLevel {

	var cNameConfig model.LogLevel
	if strings.Contains(component, "#") {
		val := strings.Split(component, "#")
		cNameConfig.ComponentName = val[0]
		cNameConfig.PackageName = strings.ReplaceAll(val[1], "/", "#")
	} else {
		cNameConfig.ComponentName = component
		cNameConfig.PackageName = defaultPackageName
	}

	return cNameConfig
}

// createConfigManager initialize default kvstore then initialize ConfigManager to connect to kvstore
func createConfigManager() (*config.ConfigManager, error) {
	kv := NewDefaultKVStore()
	cm, err := setConfigManager(kv)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// This method set loglevel for components.
// For example, using below command loglevel can be set for specific component with default packageName
// voltctl loglevel set level  <componentName>
// For example, using below command loglevel can be set for specific component with specific packageName
// voltctl loglevel set level <componentName#packageName>
// For example, using below command loglevel can be set for more than one component for default package and other component for specific packageName
// voltctl loglevel set level <componentName1#packageName> <componentName2>
func (options *SetLogLevelOpts) Execute(args []string) error {
	if options.Args.Level != "" {
		if _, err := log.StringToLogLevel(options.Args.Level); err != nil {
			return fmt.Errorf("Unknown log level %s. Allowed values are <INFO>,<DEBUG>,<ERROR>,<WARN>,<FATAL>", options.Args.Level)
		}
	}

	var componentNameConfig []model.LogLevel
	if len(options.Args.Component) == 0 {
		cNameConfig := model.LogLevel{}
		cNameConfig.ComponentName = defaultComponentName
		cNameConfig.PackageName = defaultPackageName
		componentNameConfig = append(componentNameConfig, cNameConfig)
	} else {
		for _, component := range options.Args.Component {
			cNameConfig := processCommandArgs(component)
			componentNameConfig = append(componentNameConfig, cNameConfig)
		}
	}

	cm, err := createConfigManager()
	if err != nil {
		return fmt.Errorf("Unable to create configmanager %s", err)
	}

	var output []LogLevelOutput

	for _, cConfig := range componentNameConfig {

		cNameConfig := cm.InitComponentConfig(cConfig.ComponentName, config.ConfigTypeLogLevel)

		err := cNameConfig.Save(cConfig.PackageName, options.Args.Level, context.Background())
		if err != nil {
			output = append(output, LogLevelOutput{ComponentName: cConfig.ComponentName, Status: "Failure", Error: err.Error()})
		} else {
			output = append(output, LogLevelOutput{ComponentName: cConfig.ComponentName, Status: "Success"})
		}

	}

	outputFormat := CharReplacer.Replace(options.Format)
	if outputFormat == "" {
		outputFormat = GetCommandOptionWithDefault("loglevel-set", "format", DEFAULT_SETLOGLEVEL_FORMAT)
	}
	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(options.OutputAs),
		NameLimit: options.NameLimit,
		Data:      output,
	}

	GenerateOutput(&result)
	cm.Backend.Client.Close()
	return nil
}

// This method list loglevel for components.
// For example, using below command loglevel can be list for specific component
// voltctl loglevel list  <componentName>
// For example, using below command loglevel can be list for all the components with all the packageName
// voltctl loglevel list
func (options *ListLogLevelsOpts) Execute(args []string) error {

	var data []model.LogLevel
	var componentList []string

	cm, err := createConfigManager()
	if err != nil {
		return fmt.Errorf("Unable to create configmanager %s", err)
	}

	if len(options.Args.Component) == 0 {
		componentList, err = cm.RetrieveComponentList(config.ConfigTypeLogLevel)
		if err != nil {
			return fmt.Errorf("Unable to list components %s ", err)
		}
	} else {
		componentList = append(componentList, options.Args.Component[0])
	}

	globalConfig := cm.InitComponentConfig(defaultComponentName, config.ConfigTypeLogLevel)
	globalLevel, _ := listGlobalConfig(globalConfig)

	for _, componentName := range componentList {
		cNameConfig := cm.InitComponentConfig(componentName, config.ConfigTypeLogLevel)

		componentLogConfig, err := listComponentNameConfig(cNameConfig, globalLevel)
		if err != nil {
			return fmt.Errorf("Unable to list components log level %s", err)
		}

		for packageName, level := range componentLogConfig {
			logLevel := model.LogLevel{}
			if _, err := log.StringToLogLevel(level); err != nil || packageName == "" {
				delete(componentLogConfig, packageName)
				continue
			}

			pName := strings.ReplaceAll(packageName, "#", "/")
			logLevel.PopulateFrom(componentName, pName, level)
			data = append(data, logLevel)
		}
	}

	outputFormat := CharReplacer.Replace(options.Format)
	if outputFormat == "" {
		outputFormat = GetCommandOptionWithDefault("loglevel-list", "format", DEFAULT_LOGLEVELS_FORMAT)
	}
	orderBy := options.OrderBy
	if orderBy == "" {
		orderBy = GetCommandOptionWithDefault("loglevel-list", "order", "a")
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		Filter:    options.Filter,
		OrderBy:   orderBy,
		OutputAs:  toOutputType(options.OutputAs),
		NameLimit: options.NameLimit,
		Data:      data,
	}
	GenerateOutput(&result)
	cm.Backend.Client.Close()
	return nil
}

// This method clear loglevel for components.
// For example, using below command loglevel can be clear for specific component with default packageName
// voltctl loglevel clear  <componentName>
// For example, using below command loglevel can be clear for specific component with specific packageName
// voltctl loglevel clear <componentName#packageName>
func (options *ClearLogLevelsOpts) Execute(args []string) error {

	var cConfig model.LogLevel
	if len(options.Args.Component) == 0 {
		cConfig.ComponentName = defaultComponentName
		cConfig.PackageName = defaultPackageName
	} else {
		cConfig = processCommandArgs(options.Args.Component[0])
	}

	cm, err := createConfigManager()
	if err != nil {
		return fmt.Errorf("Unable to create configmanager %s", err)
	}

	var output []LogLevelOutput

	cNameConfig := cm.InitComponentConfig(cConfig.ComponentName, config.ConfigTypeLogLevel)
	err = cNameConfig.Delete(cConfig.PackageName, context.Background())
	if err != nil {
		output = append(output, LogLevelOutput{ComponentName: cConfig.ComponentName, Status: "Failure", Error: err.Error()})
	} else {
		output = append(output, LogLevelOutput{ComponentName: cConfig.ComponentName, Status: "Success"})
	}

	outputFormat := CharReplacer.Replace(options.Format)
	if outputFormat == "" {
		outputFormat = GetCommandOptionWithDefault("loglevel-clear", "format", DEFAULT_SETLOGLEVEL_FORMAT)
	}

	result := CommandResult{
		Format:    format.Format(outputFormat),
		OutputAs:  toOutputType(options.OutputAs),
		NameLimit: options.NameLimit,
		Data:      output,
	}

	GenerateOutput(&result)
	cm.Backend.Client.Close()
	return nil
}
