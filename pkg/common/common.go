package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	k8sReport "github.com/khulnasoft/tunnel/pkg/k8s/report"
	"github.com/khulnasoft/tunnel/pkg/types"

	"golang.org/x/exp/slices"
)

var ErrorJsonUnknownField = errors.New("json: unknown field")

func IsHelp() bool {
	return slices.Contains(os.Args, "--help") || slices.Contains(os.Args, "-h")
}

func ReadReport(fileName string) (*types.Report, error) {

	log.Println("Read report", fileName)

	report, err := readAndParseJson[types.Report](fileName)
	if err == nil {
		return report, nil
	}

	if err != ErrorJsonUnknownField {
		return nil, fmt.Errorf("failed to read report %v", err)
	}

	k8s, err := readAndParseJson[k8sReport.Report](fileName)
	if err == nil {
		return convertK8sReportToReport(k8s), nil
	}

	return nil, fmt.Errorf("failed to read report %v", err)
}

func readAndParseJson[T any](fileName string) (*T, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer func() {
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("failed to close file: %w", cerr)
			}
		}()

	return parseJsonStrict[T](f)
}

func parseJsonStrict[T any](r io.Reader) (*T, error) {
	var out T

	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&out); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			return nil, ErrorJsonUnknownField
		}
		return nil, err
	}

	return &out, nil
}

func convertK8sReportToReport(k8s *k8sReport.Report) *types.Report {
    var results types.Results
    if k8s != nil {
        for _, vuln := range k8s.Vulnerabilities {
            if vuln.Results != nil {
                results = append(results, vuln.Results...)
            }
        }
        for _, misc := range k8s.Misconfigurations {
            if misc.Results != nil {
                results = append(results, misc.Results...)
            }
        }
    }

func GetPathToPluginDir(fileName string) (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(ex), fileName), nil
}

func GetPathToTemplate(fileName string) (string, error) {
	path, err := GetPathToPluginDir(fileName)
	if err != nil {
		return "", err
	}
	return "@" + path, nil
}

func ReadPluginFile(fileName string) ([]byte, error) {
	path, err := GetPathToPluginDir(fileName)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path)
}

import "context"

func MakeTunnelJsonReport(ctx context.Context, tunnelCommand []string, outputFileName string) error {
	cmdArgs := append(tunnelCommand, "--format", "json", "--output", outputFileName)
	cmd := exec.CommandContext(ctx, "tunnel", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run tunnel: %w", err)
	}
	return nil
}

type Arguments map[string]string

func RetrievePluginArguments(availableArguments []string) (pluginArgs Arguments, rest []string) {
	tunnelCommand := make([]string, 0, len(os.Args))
	args := make(map[string]string)
	for i := 0; i < len(os.Args); i++ {
		if slices.Contains(availableArguments, os.Args[i]) {
			if i+1 >= len(os.Args) {
				args[os.Args[i]] = ""
			} else {
				args[os.Args[i]] = os.Args[i+1]
			}
			i++ // skip argument value
		} else {
			tunnelCommand = append(tunnelCommand, os.Args[i])
		}
	}
	return args, tunnelCommand[1:]
}
