package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netlab/netlab/internal/compliance"
)

func main() {
	if len(os.Args) < 2 {
		usage(2)
	}
	var err error
	switch os.Args[1] {
	case "validate":
		err = validate(os.Args[2:])
	case "report":
		err = report(os.Args[2:])
	case "capture-candidate":
		err = captureCandidate(os.Args[2:])
	case "scan-evidence":
		err = scanEvidence(os.Args[2:])
	default:
		usage(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(compliance.ExitCode(err))
	}
}

func usage(code int) {
	fmt.Fprintln(os.Stderr, "usage: netlab-compliance <validate|report|capture-candidate|scan-evidence>")
	os.Exit(code)
}

func scanEvidence(arguments []string) error {
	set := flag.NewFlagSet("scan-evidence", flag.ContinueOnError)
	directory := set.String("directory", "compliance/evidence", "evidence directory")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	count := 0
	err := filepath.WalkDir(*directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		count += len(compliance.ScanEvidence(path, body))
		return nil
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"prohibited_content_count": count})
}

func commonFlags(name string, arguments []string) (*flag.FlagSet, *string, *string, *string, *string) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	ledger := set.String("ledger", "compliance/constitution-ledger.json", "ledger path")
	deployment := set.String("deployment", "compliance/deployment-authority.json", "deployment inventory path")
	templates := set.String("templates", "compliance/template-readiness.json", "template readiness path")
	evidence := set.String("evidence-dir", "compliance/evidence", "evidence directory")
	_ = set.Parse(arguments)
	return set, ledger, deployment, templates, evidence
}

func validate(arguments []string) error {
	set, ledger, deployment, templates, evidence := commonFlags("validate", arguments)
	if err := set.Parse(set.Args()); err != nil {
		return err
	}
	documents, err := compliance.LoadDocuments(*ledger, *deployment, *templates, *evidence)
	if err != nil {
		return err
	}
	return compliance.ValidateDocuments(documents)
}

func report(arguments []string) error {
	set := flag.NewFlagSet("report", flag.ContinueOnError)
	ledgerPath := set.String("ledger", "compliance/constitution-ledger.json", "ledger path")
	acceptancePath := set.String("acceptance", "", "acceptance run path")
	jsonOutput := set.Bool("json", false, "write JSON")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	var ledger compliance.Ledger
	body, err := os.ReadFile(*ledgerPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &ledger); err != nil {
		return err
	}
	if *acceptancePath != "" {
		var run compliance.AcceptanceRun
		body, err = os.ReadFile(*acceptancePath)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(body, &run); err != nil {
			return err
		}
		if err = compliance.ValidateReportConsistency(ledger, run); err != nil {
			return err
		}
	}
	return compliance.WriteReport(os.Stdout, ledger, *jsonOutput)
}

func captureCandidate(arguments []string) error {
	set := flag.NewFlagSet("capture-candidate", flag.ContinueOnError)
	version := set.String("version", "dev", "release version")
	candidateID := set.String("candidate-id", "", "candidate id")
	binary := set.String("binary", "", "binary path")
	contracts := set.String("contracts", "specs/002-constitution-gap-closure/contracts", "contracts directory")
	output := set.String("output", "-", "output path or -")
	if err := set.Parse(arguments); err != nil {
		return err
	}
	if *candidateID == "" {
		return fmt.Errorf("candidate-id required")
	}
	identity, err := compliance.CaptureCandidate(*version, *candidateID, *binary, *contracts, time.Now().UTC())
	if err != nil {
		return err
	}
	body, _ := json.MarshalIndent(identity, "", "  ")
	body = append(body, '\n')
	if *output == "-" {
		_, err = os.Stdout.Write(body)
		return err
	}
	return os.WriteFile(*output, body, 0o600)
}
