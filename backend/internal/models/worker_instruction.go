package models

type WorkerInstructionField string

const (
	WorkerInstructionFieldJob     WorkerInstructionField = "job"
	WorkerInstructionFieldPlan    WorkerInstructionField = "plan"
	WorkerInstructionFieldDiscuss WorkerInstructionField = "discuss"
)

func IsWorkerInstructionField(value string) bool {
	switch WorkerInstructionField(value) {
	case WorkerInstructionFieldJob, WorkerInstructionFieldPlan, WorkerInstructionFieldDiscuss:
		return true
	default:
		return false
	}
}
