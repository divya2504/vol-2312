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
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/opencord/voltctl/pkg/format"
	"github.com/opencord/voltctl/pkg/model"
	"github.com/opencord/voltha-lib-go/v3/pkg/config"
	"github.com/opencord/voltha-lib-go/v3/pkg/log"
	"strings"
)

type LogLevelOutput struct {
	ComponentName string
	Status        string
	Error         string
}

type SetLogLevelOpts struct {
	OutputOptions
	Args struct {
		Level     string
		Component []string
	} `positional-args:"yes" required:"yes"`
}

type ListLogLevelsOpts struct {
	ListOutputOptions
	Args struct {
		Component []string
	} `positional-args:"yes" required:"yes"`
}

type ClearLogLevelsOpts struct {
	OutputOptions
	Args struct {
		Component []string
	} `positional-args:"yes" required:"yes"`
}

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

func RegisterLogLevelCommands(parent *flags.Parser) {
	_, err := parent.AddCommand("loglevel", "loglevel commands", "Get and set log levels", &logLevelOpts)
	if err != nil {
		Error.Fatalf("Unable to register log level commands with voltctl command parser: %s", err.Error())
	}
}

type componentLogController struct {
	componentNameConfig *config.ComponentConfig
	listConfig          *config.ComponentConfig
	globalConfig        *config.ComponentConfig
	configManager       *config.ConfigManager
}

func NewComponentLogController(cm *config.ConfigManager) *componentLogController {

	clc := &componentLogController{
		componentNameConfig: nil,
		listConfig:          nil,
		globalConfig:        nil,
		configManager:       cm}

	return clc
}

func createComponentConfig(cm *config.ConfigManager, cc *componentLogController, componentName string) {
	if componentName == "list" {
		cc.listConfig = cm.InitComponentConfig(componentName, config.ConfigTypeLogLevel)
	} else {
		cc.globalConfig = cm.InitComponentConfig("global", config.ConfigTypeLogLevel)
		cc.componentNameConfig = cm.InitComponentConfig(componentName, config.ConfigTypeLogLevel)
	}
}

func (c *componentLogController) setGlobalConfig(packageName, level string) error {
	err := c.globalConfig.Save(packageName, level)
	if err != nil {
		return err
	}

	return nil
}

func (c *componentLogController) setComponentNameConfig(packageName, level string) error {
	err := c.componentNameConfig.Save(packageName, level)
	if err != nil {
		return err
	}
	return nil
}

func (c *componentLogController) listAllComponentConfig() ([]config.List, error) {
	configList, err := c.listConfig.RetrieveList()
	if err != nil {
		return nil, err
	}

	return configList, nil
}

func (c *componentLogController) listGlobalConfig() (string, error) {
	var globalDefaultLogLevel string
	globalLogConfig, err := c.globalConfig.RetrieveAll()
	if err != nil {
		return "", err
	}

	for globalKey, globalLevel := range globalLogConfig {
		if globalKey == "default" {
			globalDefaultLogLevel = globalLevel
		}
	}

	return globalDefaultLogLevel, nil
}

func (c *componentLogController) listComponentNameConfig() (map[string]string, error) {
	globalLevel, err := c.listGlobalConfig()
	if err != nil {
		return nil, err
	}

	componentLogConfig, err := c.componentNameConfig.RetrieveAll()
	if err != nil {
		return nil, err
	}

	if globalLevel != "" {
		componentLogConfig["default"] = globalLevel
	}
	return componentLogConfig, nil
}

func (c *componentLogController) clearGlobalConfig(packageName string) error {
	err := c.globalConfig.Delete(packageName)
	if err != nil {
		return err
	}

	return nil
}

func (c *componentLogController) clearComponentNameConfig(packageName string) error {
	err := c.componentNameConfig.Delete(packageName)
	if err != nil {
		return err
	}
	return nil
}

func processCommandArgs(components []string) map[string]string {
	componentConfig := make(map[string]string)

	for _, componentKey := range components {
		if strings.Contains(componentKey, "#") {
			val := strings.Split(componentKey, "#")
			componentName := val[0]
			packageName := strings.ReplaceAll(val[1], "/", "#")
			componentConfig[componentName] = packageName
		} else {
			componentConfig[componentKey] = "default"
		}

	}
	return componentConfig
}

func (options *SetLogLevelOpts) Execute(args []string) error {
	if options.Args.Level != "" {
		if _, err := log.StringToLogLevel(options.Args.Level); err != nil {
			return fmt.Errorf("Please specify valid logLevel")
		}
	}

	componentConfig := make(map[string]string)
	if len(options.Args.Component) == 0 {
		componentConfig["global"] = "default"
	} else {
		componentConfig = processCommandArgs(options.Args.Component)
	}

	kv := NewKVStore()
	cm, err := kv.setKVClient()
	if err != nil {
		return err
	}
	cc := NewComponentLogController(cm)

	var output []LogLevelOutput

	for componentName, packageName := range componentConfig {

		createComponentConfig(cm, cc, componentName)
		if componentName == "global" {
			err := cc.setGlobalConfig(packageName, options.Args.Level)
			if err != nil {
				output = append(output, LogLevelOutput{ComponentName: componentName, Status: "Failure", Error: err.Error()})
			}
		} else {
			err := cc.setComponentNameConfig(packageName, options.Args.Level)
			if err != nil {
				output = append(output, LogLevelOutput{ComponentName: componentName, Status: "Failure", Error: err.Error()})
			}
		}

		output = append(output, LogLevelOutput{ComponentName: componentName, Status: "Success"})

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
	return nil
}

func (options *ListLogLevelsOpts) listLogLevels(methodName string, args []string) error {
	if len(options.Args.Component) == 2 {
		return fmt.Errorf("Please don't specify more than one component name")
	}

	var data []model.LogLevel
	kv := NewKVStore()
	cm, err := kv.setKVClient()
	if err != nil {
		return err
	}
	cc := NewComponentLogController(cm)

	if len(options.Args.Component) == 0 {
		createComponentConfig(cm, cc, "list")
		componentLogConfig, err := cc.listAllComponentConfig()
		if err != nil {
			return err
		}
		for _, l := range componentLogConfig {
			logLevel := model.LogLevel{}
			pName := strings.ReplaceAll(l.PackageName, "#", "/")
			logLevel.PopulateFrom(l.ComponentName, pName, l.Level)
			data = append(data, logLevel)
		}
	} else {
		componentName := options.Args.Component[0]

		createComponentConfig(cm, cc, componentName)

		componentLogConfig, err := cc.listComponentNameConfig()
		if err != nil {
			return err
		}

		for packageName, level := range componentLogConfig {
			logLevel := model.LogLevel{}
			pName := strings.ReplaceAll(packageName, "#", "/")
			logLevel.PopulateFrom(componentName, pName, level)
			data = append(data, logLevel)
		}

	}
	outputFormat := CharReplacer.Replace(options.Format)
	if outputFormat == "" {
		outputFormat = GetCommandOptionWithDefault(methodName, "format", DEFAULT_LOGLEVELS_FORMAT)
	}
	orderBy := options.OrderBy
	if orderBy == "" {
		orderBy = GetCommandOptionWithDefault(methodName, "order", "")
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
	return nil
}

func (options *ListLogLevelsOpts) Execute(args []string) error {
	return options.listLogLevels("loglevel-list", args)
}

func (options *ClearLogLevelsOpts) Execute(args []string) error {
	componentConfig := make(map[string]string)
	if len(options.Args.Component) == 0 {
		componentConfig["global"] = "default"
	} else {
		componentConfig = processCommandArgs(options.Args.Component)
	}

	kv := NewKVStore()
	cm, err := kv.setKVClient()
	if err != nil {
		return err
	}
	cc := NewComponentLogController(cm)

	var output []LogLevelOutput

	for componentName, packageName := range componentConfig {

		createComponentConfig(cm, cc, componentName)
		if componentName == "global" {
			err := cc.clearGlobalConfig(packageName)
			if err != nil {
				output = append(output, LogLevelOutput{ComponentName: componentName, Status: "Failure", Error: err.Error()})
			}
		} else {
			err := cc.clearComponentNameConfig(packageName)
			if err != nil {
				output = append(output, LogLevelOutput{ComponentName: componentName, Status: "Failure", Error: err.Error()})
			}
		}

		output = append(output, LogLevelOutput{ComponentName: componentName, Status: "Success"})

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
	return nil
}
