package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-steputils/tools"
)

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func packageCodeCoveragePath() (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("go-test")
	if err != nil {
		return "", fmt.Errorf("Failed to create tmp dir for code coverage reports: %s", err)
	}
	return filepath.Join(tmpDir, "profile.out"), nil
}

func codeCoveragePath() (string, error) {
	deployDir := os.Getenv("BITRISE_DEPLOY_DIR")
	if deployDir == "" {
		return "", fmt.Errorf("BITRISE_DEPLOY_DIR env not set")
	}
	if err := os.MkdirAll(deployDir, 0777); err != nil {
		return "", fmt.Errorf("Failed to create BITRISE_DEPLOY_DIR: %s", err)
	}
	return filepath.Join(deployDir, "go_code_coverage.txt"), nil
}

func appendPackageCoverageAndDelete(packageCoveragePth, coveragePth string) error {
	content, err := fileutil.ReadStringFromFile(packageCoveragePth)
	if err != nil {
		return fmt.Errorf("Failed to read package code coverage report: %s", err)
	}

	if err := fileutil.AppendStringToFile(coveragePth, content); err != nil {
		return fmt.Errorf("Failed to append package code coverage report: %s", err)
	}

	if err := os.RemoveAll(packageCoveragePth); err != nil {
		return fmt.Errorf("Failed to remove package code coverage report: %s", err)
	}
	return nil
}

func main() {
	packages := os.Getenv("packages")

	log.Infof("Configs:")
	log.Printf("- packages: %s", packages)

	if packages == "" {
		failf("Required input not defined: packages")
	}

	log.Infof("\nRunning go test...")

	packageCodeCoveragePth, err := packageCodeCoveragePath()
	if err != nil {
		failf(err.Error())
	}

	codeCoveragePth, err := codeCoveragePath()
	if err != nil {
		failf(err.Error())
	}

	for _, p := range strings.Split(packages, "\n") {
		cmd := command.NewWithStandardOuts("go", "test", "-v", "-race", "-coverprofile="+packageCodeCoveragePth, "-covermode=atomic", p)

		log.Printf("$ %s", cmd.PrintableCommandArgs())

		if err := cmd.Run(); err != nil {
			failf("go test failed: %s", err)
		}

		if err := appendPackageCoverageAndDelete(packageCodeCoveragePth, codeCoveragePth); err != nil {
			failf(err.Error())
		}
	}

	if err := tools.ExportEnvironmentWithEnvman("GO_CODE_COVERAGE_REPORT_PATH", codeCoveragePth); err != nil {
		failf("Failed to export GO_CODE_COVERAGE_REPORT_PATH=%s", codeCoveragePth)
	}

	log.Donef("\ncode coverage is available at: GO_CODE_COVERAGE_REPORT_PATH=%s", codeCoveragePth)
}
