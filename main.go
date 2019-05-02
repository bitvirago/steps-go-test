package main

import (
	"fmt"
	"os"
	"path/filepath"

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

func createPackageCodeCoverageFile() (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("go-test-coverage")
	if err != nil {
		return "", fmt.Errorf("Failed to create tmp dir for code coverage reports: %s", err)
	}
	pth := filepath.Join(tmpDir, "coverprofile.out")
	if _, err := os.Create(pth); err != nil {
		return "", err
	}
	return pth, nil
}

func codeCoveragePath() (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("go-test-coverage")
	if err != nil {
		return "", fmt.Errorf("Failed to create tmp dir for code coverage reports: %s", err)
	}
	pth := filepath.Join(tmpDir, "go_code_coverage.txt")
	if _, err := os.Create(pth); err != nil {
		return "", err
	}
	return pth, nil
}

func appendPackageCoverageAndRecreate(packageCoveragePth, coveragePth string) error {
	content, err := fileutil.ReadStringFromFile(packageCoveragePth)
	if err != nil {
		return fmt.Errorf("Failed to read package code coverage report file: %s", err)
	}

	if err := fileutil.AppendStringToFile(coveragePth, content); err != nil {
		return fmt.Errorf("Failed to append package code coverage report: %s", err)
	}

	if err := os.RemoveAll(packageCoveragePth); err != nil {
		return fmt.Errorf("Failed to remove package code coverage report file: %s", err)
	}
	if _, err := os.Create(packageCoveragePth); err != nil {
		return fmt.Errorf("Failed to create package code coverage report file: %s", err)
	}
	return nil
}

func main() {
	log.Infof("\nRunning go test...")

	packageCodeCoveragePth, err := createPackageCodeCoverageFile()
	if err != nil {
		failf(err.Error())
	}

	codeCoveragePth, err := codeCoveragePath()
	if err != nil {
		failf(err.Error())
	}

	cmd := command.NewWithStandardOuts("go", "test", "-v", "-race", "-coverprofile="+packageCodeCoveragePth, "-covermode=atomic", "./...")

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("go test failed: %s", err)
	}

	cmd := command.NewWithStandardOuts("go", "tool", "cover", "-html", packageCodeCoveragePth, "-o", "go_code_coverage.html")

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("go test failed: %s", err)
	}

	if err := appendPackageCoverageAndRecreate(packageCodeCoveragePth, codeCoveragePth); err != nil {
		failf(err.Error())
	}

	if err := tools.ExportEnvironmentWithEnvman("GO_CODE_COVERAGE_REPORT_PATH", codeCoveragePth); err != nil {
		failf("Failed to export GO_CODE_COVERAGE_REPORT_PATH=%s", codeCoveragePth)
	}

	log.Donef("\ncode coverage is available at: GO_CODE_COVERAGE_REPORT_PATH=%s", codeCoveragePth)
}
