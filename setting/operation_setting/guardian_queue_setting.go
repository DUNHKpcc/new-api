package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// GuardianQueueSetting lets administrators pace hands-on Guardian deliveries.
type GuardianQueueSetting struct {
	DailyCapacity int   `json:"daily_capacity"` // orders started per day; 0 means no limit
	ExtraSlots    int   `json:"extra_slots"`    // placeholder jobs ahead of every queued buyer
	Paused        bool  `json:"paused"`
	ResumeAt      int64 `json:"resume_at"` // Unix seconds; 0 pauses until resumed manually
}

var guardianQueueSetting = GuardianQueueSetting{}

func init() {
	config.GlobalConfig.Register("guardian_queue", &guardianQueueSetting)
}

func GetGuardianQueueSetting() GuardianQueueSetting {
	return guardianQueueSetting
}

// PausedAt reports whether the queue is paused at the given Unix time; a passed resume time ends the pause.
func (s GuardianQueueSetting) PausedAt(now int64) bool {
	return s.Paused && (s.ResumeAt == 0 || s.ResumeAt > now)
}
