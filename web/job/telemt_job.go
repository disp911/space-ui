package job

import (
	"github.com/disp911/space-ui/v3/web/service"
)

// TelemtJob keeps the telemt proxies behind mtproto inbounds in line with
// the database and records their traffic.
type TelemtJob struct {
	telemtService service.TelemtService
}

// NewTelemtJob creates a new telemt sync and traffic job.
func NewTelemtJob() *TelemtJob {
	return new(TelemtJob)
}

// Run syncs the proxies (restarting any that exited) and collects traffic.
func (j *TelemtJob) Run() {
	j.telemtService.Sync()
	j.telemtService.CollectTraffic()
}
