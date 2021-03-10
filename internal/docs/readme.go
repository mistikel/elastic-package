// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package docs

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-package/internal/logger"
	"github.com/elastic/elastic-package/internal/packages"
)

// IsReadmeUpToDate function checks if all .md readme files are up-to-date.
func IsReadmeUpToDate() (bool, error) {
	packageRoot, err := packages.MustFindPackageRoot()
	if err != nil {
		return false, errors.Wrap(err, "package root not found")
	}

	readmeFiles, err := ioutil.ReadDir(filepath.Join(packageRoot, "_dev", "build", "docs"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to return a list of directory entries from %s", packageRoot)
	}

	for _, readme := range readmeFiles {
		filename := readme.Name()
		logger.Debugf("Check if %s is up-to-date", filename)

		rendered, shouldBeRendered, err := generateReadme(packageRoot, filename)
		if err != nil {
			return false, err
		}
		if !shouldBeRendered {
			continue // This readme file is static and doesn't use template.
		}

		existing, found, err := readReadme(packageRoot, filename)
		if err != nil {
			return false, errors.Wrapf(err, "reading %s file failed", filename)
		}
		if !found {
			return false, nil
		}

		if !bytes.Equal(existing, rendered) {
			return false, nil
		}
	}
	return true, nil
}

// UpdateReadme function updates all .md readme files using ą defined template files. The function doesn't perform any action
// if the template file is not present.
func UpdateReadme() (map[string]string, error) {
	packageRoot, err := packages.MustFindPackageRoot()
	if err != nil {
		return nil, errors.Wrap(err, "package root not found")
	}

	readmeFiles, err := ioutil.ReadDir(filepath.Join(packageRoot, "_dev", "build", "docs"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to return a list of directory entries from %s", packageRoot)
	}

	targets := map[string]string{}
	for _, readme := range readmeFiles {
		filename := readme.Name()
		logger.Debugf("Update the %s file", filename)

		rendered, shouldBeRendered, err := generateReadme(packageRoot, filename)
		if err != nil {
			return nil, err
		}
		if !shouldBeRendered {
			return nil, nil
		}

		target, err := writeReadme(filename, packageRoot, rendered)
		if err != nil {
			return nil, errors.Wrapf(err, "writing %s file failed", filename)
		}

		targets[filename] = target
	}
	return targets, nil
}

func generateReadme(packageRoot string, readmeFileName string) ([]byte, bool, error) {
	logger.Debugf("Generate %s file (package: %s)", readmeFileName, packageRoot)
	templatePath, found, err := findReadmeTemplatePath(packageRoot, readmeFileName)
	if err != nil {
		return nil, false, errors.Wrapf(err, "can't locate %s template file", readmeFileName)
	}
	if !found {
		logger.Debug("README file is static, can't be generated from the template file")
		return nil, false, nil
	}
	logger.Debugf("Template file for %s found: %s", readmeFileName, templatePath)

	rendered, err := renderReadme(readmeFileName, packageRoot, templatePath)
	if err != nil {
		return nil, true, errors.Wrap(err, "rendering Readme failed")
	}
	return rendered, true, nil
}

func findReadmeTemplatePath(packageRoot, readmeFileName string) (string, bool, error) {
	templatePath := filepath.Join(packageRoot, "_dev", "build", "docs", readmeFileName)
	_, err := os.Stat(templatePath)
	if err != nil && os.IsNotExist(err) {
		return "", false, nil // README.md file not found
	}
	if err != nil {
		return "", false, errors.Wrapf(err, "can't located the %s file", readmeFileName)
	}
	return templatePath, true, nil
}

func renderReadme(readmeFileName, packageRoot, templatePath string) ([]byte, error) {
	logger.Debugf("Render %s file (package: %s, templatePath: %s)", readmeFileName, packageRoot, templatePath)

	t := template.New(readmeFileName)
	t, err := t.Funcs(template.FuncMap{
		"event": func(dataStreamName string) (string, error) {
			return renderSampleEvent(packageRoot, dataStreamName)
		},
		"fields": func(dataStreamName string) (string, error) {
			return renderExportedFields(packageRoot, dataStreamName)
		},
	}).ParseFiles(templatePath)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing README template failed (path: %s)", templatePath)
	}

	var rendered bytes.Buffer
	err = t.Execute(&rendered, nil)
	if err != nil {
		return nil, errors.Wrap(err, "executing template failed")
	}
	return rendered.Bytes(), nil
}

func readReadme(packageRoot, readmeFileName string) ([]byte, bool, error) {
	logger.Debugf("Read existing %s file (package: %s)", readmeFileName, packageRoot)

	readmePath := filepath.Join(packageRoot, "docs", readmeFileName)
	b, err := ioutil.ReadFile(readmePath)
	if err != nil && os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errors.Wrapf(err, "readfile failed (path: %s)", readmePath)
	}
	return b, true, err
}

func writeReadme(readmeFileName, packageRoot string, content []byte) (string, error) {
	logger.Debugf("Write %s file (package: %s)", readmeFileName, packageRoot)

	docsPath := docsPath(packageRoot)
	logger.Debugf("Create directories: %s", docsPath)
	err := os.MkdirAll(docsPath, 0755)
	if err != nil {
		return "", errors.Wrapf(err, "mkdir failed (path: %s)", docsPath)
	}

	aReadmePath := readmePath(packageRoot, readmeFileName)
	logger.Debugf("Write %s file to: %s", readmeFileName, aReadmePath)

	err = ioutil.WriteFile(aReadmePath, content, 0644)
	if err != nil {
		return "", errors.Wrapf(err, "writing file failed (path: %s)", aReadmePath)
	}
	return aReadmePath, nil
}

func readmePath(packageRoot, readmeFileName string) string {
	return filepath.Join(docsPath(packageRoot), readmeFileName)
}

func docsPath(packageRoot string) string {
	return filepath.Join(packageRoot, "docs")
}
