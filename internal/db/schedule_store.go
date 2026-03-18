package db

import (
	"sync"

	"faas-engine-go/internal/types"
)

var (
	scheduleStore = make(map[string]types.Schedule)
	scheduleMu    sync.Mutex
)

func Reset() {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()
	scheduleStore = make(map[string]types.Schedule)
}

// CREATE
func AddSchedule(s types.Schedule) {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	scheduleStore[s.ID] = s
}

// DELETE
func DeleteSchedule(id string) {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	delete(scheduleStore, id)
}

// GET ONE
func GetSchedule(id string) (types.Schedule, bool) {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	s, ok := scheduleStore[id]
	return s, ok
}

// LIST ALL
func ListSchedules() []types.Schedule {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	var result []types.Schedule
	for _, s := range scheduleStore {
		result = append(result, s)
	}
	return result
}

func GetSchedulesByFunction(functionName string) []types.Schedule {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	var result []types.Schedule

	for _, sch := range scheduleStore {
		if sch.FunctionName == functionName {
			result = append(result, sch)
		}
	}

	return result
}
