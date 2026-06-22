package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"isobox/internal/policy"
)

const (
	taskRecordSchemaVersion = "v1"
)

var supportedTaskRecordSchemaVersions = map[string]struct{}{
	taskRecordSchemaVersion: {},
}

func loadRecord(recordDir string) (taskRecord, error) {
	var record taskRecord

	recordPath := filepath.Join(recordDir, "record.json")
	recordBytes, err := os.ReadFile(recordPath)
	if err != nil {
		return record, fmt.Errorf("read task record %s: %w", recordPath, err)
	}

	if err := json.Unmarshal(recordBytes, &record); err != nil {
		return record, fmt.Errorf("parse task record %s: %w", recordPath, err)
	}

	if err := validateTaskRecord(record); err != nil {
		return record, fmt.Errorf("validate task record %s: %w", recordPath, err)
	}

	return record, nil
}

func validateTaskRecord(record taskRecord) error {
	if record.SchemaVersion == "" {
		return errors.New("missing required field: schema_version")
	}
	if _, ok := supportedTaskRecordSchemaVersions[record.SchemaVersion]; !ok {
		return fmt.Errorf("unsupported schema_version %q (isobox supports %s); upgrade isobox to load this record",
			record.SchemaVersion, supportedTaskRecordSchemaVersionList())
	}
	if record.ID == "" {
		return errors.New("missing required field: id")
	}
	if record.CreatedAt == "" {
		return errors.New("missing required field: created_at")
	}
	if err := validateEffectivePolicy(record.EffectivePolicy); err != nil {
		return err
	}
	if err := validateOutcome(record.Outcome); err != nil {
		return err
	}
	return nil
}

func validateEffectivePolicy(p effectivePolicy) error {
	if p.SchemaVersion == "" {
		return errors.New("effective_policy missing required field: schema_version")
	}
	if p.WorkspaceSource == "" {
		return errors.New("effective_policy missing required field: workspace_source")
	}
	if len(p.WorkloadCommand) == 0 {
		return errors.New("effective_policy missing required field: workload_command")
	}
	if p.RuntimeBackend == "" {
		return errors.New("effective_policy missing required field: runtime_backend")
	}
	if p.RetentionDefault == "" {
		return errors.New("effective_policy missing required field: retention_default")
	}
	if err := validateReuseInputs(p.ReuseInputs); err != nil {
		return err
	}
	return nil
}

func validateReuseInputs(inputs []policy.ReuseInput) error {
	for i, input := range inputs {
		if err := policy.ValidateReuseInputKind(string(input.Kind)); err != nil {
			return fmt.Errorf("effective_policy reuse_inputs[%d]: %w", i, err)
		}
		if input.Value == "" {
			return fmt.Errorf("effective_policy reuse_inputs[%d] (%s): empty value", i, input.Kind)
		}
	}
	return nil
}

func validateOutcome(o taskAttemptOutcome) error {
	if o.Type == "" {
		return errors.New("outcome missing required field: type")
	}
	switch o.Type {
	case outcomeSuccess, outcomePreparationFailure, outcomeLaunchFailure,
		outcomeWorkloadCommandExit, outcomeResultCaptureFailure:
		return nil
	default:
		return fmt.Errorf("outcome has unsupported type %q", o.Type)
	}
}

func supportedTaskRecordSchemaVersionList() string {
	versions := make([]string, 0, len(supportedTaskRecordSchemaVersions))
	for version := range supportedTaskRecordSchemaVersions {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return strings.Join(versions, ", ")
}
