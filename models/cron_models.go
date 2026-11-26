package models

import "time"

// CronJob merepresentasikan 1 entri pekerjaan terjadwal di tabel cron_jobs
type CronJob struct {
	ID              int        `json:"id" gorm:"primaryKey;autoIncrement"`
	JobKey          string     `json:"job_key" gorm:"column:job_key;size:50;unique;not null"`
	Nama            string     `json:"nama" gorm:"column:nama;size:100;not null"`
	Desc            string     `json:"desc" gorm:"column:desc;type:text"`
	CronExpr        string     `json:"cron_expr" gorm:"column:cron_expr;size:50;not null;default:'0 6-23,0 * * *'"`
	IsActive        int        `json:"is_active" gorm:"column:is_active;not null;default:1"`
	IsRunning       int        `json:"is_running" gorm:"column:is_running;not null;default:0"`
	LastRunAt       *time.Time `json:"last_run_at" gorm:"column:last_run_at"`
	LastStatus      string     `json:"last_status" gorm:"column:last_status"`
	LastDurationSec int        `json:"last_duration_sec" gorm:"column:last_duration_sec;default:0"`
	LastMessage     string     `json:"last_message" gorm:"column:last_message;type:text"`
	NotifyWA        int        `json:"notify_wa" gorm:"column:notify_wa;not null;default:0"`
	Created         *time.Time `json:"created" gorm:"column:created;autoCreateTime"`
	Updated         *time.Time `json:"updated" gorm:"column:updated;autoUpdateTime"`
}

func (CronJob) TableName() string {
	return "cron_jobs"
}

// ScheduleTaskLog merepresentasikan entri audit eksekusi di tabel zz_schedule_task
type ScheduleTaskLog struct {
	ID          int        `json:"id" gorm:"primaryKey;autoIncrement"`
	Nama        string     `json:"nama" gorm:"column:nama;size:50;not null"`
	Flag        int        `json:"flag" gorm:"column:flag;not null;default:0"`
	Jml         int        `json:"jml" gorm:"column:jml;not null;default:0"`
	RunningTime *time.Time `json:"running_time" gorm:"column:running_time"`
	FinishTime  *time.Time `json:"finish_time" gorm:"column:finish_time"`
}

func (ScheduleTaskLog) TableName() string {
	return "zz_schedule_task"
}

// UpdateCronJobRequest payload untuk memperbarui konfigurasi job
type UpdateCronJobRequest struct {
	Nama     string `json:"nama"`
	Desc     string `json:"desc"`
	CronExpr string `json:"cron_expr"`
	IsActive *int   `json:"is_active"`
	NotifyWA *int   `json:"notify_wa"`
}
