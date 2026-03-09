package services

import (
	"errors"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/models"
)

const (
	factoryInstructionJob     = "You are the assigned worker agent for this workspace.\nFollow repository instructions such as AGENTS.md, the spec, and the architecture docs when relevant.\nKeep scope tight and changes easy to review.\nDo not guess missing requirements; ask when needed.\nPoint out weak assumptions or conflicts directly.\nState important assumptions, tradeoffs, and risks clearly."
	factoryInstructionPlan    = "You are converting scoped discussion into an execution-ready prompt.\nBe explicit, concrete, and implementation-oriented.\nDo not add unnecessary scope or extra deliverables."
	factoryInstructionDiscuss = "You are helping the user scope work before execution.\nKeep the conversation focused, concrete, and efficient.\nChallenge vague requirements and surface missing constraints directly."
)

func applyFactoryWorkerInstructions(worker *models.Worker) {
	worker.InstructionJob = factoryInstructionJob
	worker.InstructionPlan = factoryInstructionPlan
	worker.InstructionDiscuss = factoryInstructionDiscuss
}

func factoryWorkerInstruction(field models.WorkerInstructionField) (string, error) {
	switch field {
	case models.WorkerInstructionFieldJob:
		return factoryInstructionJob, nil
	case models.WorkerInstructionFieldPlan:
		return factoryInstructionPlan, nil
	case models.WorkerInstructionFieldDiscuss:
		return factoryInstructionDiscuss, nil
	default:
		return "", errors.New("invalid instruction field")
	}
}

func composeWorkerInstruction(worker *models.Worker, field models.WorkerInstructionField, body string) (string, error) {
	var instruction string

	switch field {
	case models.WorkerInstructionFieldJob:
		instruction = worker.InstructionJob
	case models.WorkerInstructionFieldPlan:
		instruction = worker.InstructionPlan
	case models.WorkerInstructionFieldDiscuss:
		instruction = worker.InstructionDiscuss
	default:
		return "", errors.New("invalid instruction field")
	}

	instruction = strings.TrimSpace(instruction)
	body = strings.TrimSpace(body)

	if instruction == "" {
		return body, nil
	}
	if body == "" {
		return instruction, nil
	}
	return instruction + "\n\n" + body, nil
}
