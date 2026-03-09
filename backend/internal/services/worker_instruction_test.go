package services

import (
	"testing"

	"github.com/chatchawan/agent-orchestrator/internal/models"
)

func TestApplyFactoryWorkerInstructions(t *testing.T) {
	worker := &models.Worker{}

	applyFactoryWorkerInstructions(worker)

	if worker.InstructionJob != factoryInstructionJob {
		t.Fatalf("unexpected job instruction: %q", worker.InstructionJob)
	}
	if worker.InstructionPlan != factoryInstructionPlan {
		t.Fatalf("unexpected plan instruction: %q", worker.InstructionPlan)
	}
	if worker.InstructionDiscuss != factoryInstructionDiscuss {
		t.Fatalf("unexpected discuss instruction: %q", worker.InstructionDiscuss)
	}
}

func TestComposeWorkerInstructionUsesWorkerField(t *testing.T) {
	worker := &models.Worker{
		InstructionJob:     "job instruction",
		InstructionPlan:    "plan instruction",
		InstructionDiscuss: "discuss instruction",
	}

	jobInstruction, err := composeWorkerInstruction(worker, models.WorkerInstructionFieldJob, "fix the bug")
	if err != nil {
		t.Fatalf("compose job: %v", err)
	}
	if jobInstruction != "job instruction\n\nfix the bug" {
		t.Fatalf("unexpected job instruction: %q", jobInstruction)
	}

	discussInstruction, err := composeWorkerInstruction(worker, models.WorkerInstructionFieldDiscuss, "ask one question")
	if err != nil {
		t.Fatalf("compose discuss: %v", err)
	}
	if discussInstruction != "discuss instruction\n\nask one question" {
		t.Fatalf("unexpected discuss instruction: %q", discussInstruction)
	}

	planInstruction, err := composeWorkerInstruction(worker, models.WorkerInstructionFieldPlan, "generate the prompt")
	if err != nil {
		t.Fatalf("compose plan: %v", err)
	}
	if planInstruction != "plan instruction\n\ngenerate the prompt" {
		t.Fatalf("unexpected plan instruction: %q", planInstruction)
	}
}
